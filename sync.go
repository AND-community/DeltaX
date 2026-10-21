package main

// sync.go — grup üyeleri arasında merkezi ağdan bağımsız özel senkronizasyon.
//
// İki kanal kullanılır:
//   - DM (AND'ın /api/v1/dm/*)  → küçük kontrol mesajları (davet, push
//     bildirimi, pull isteği). Metin alanına base64 JSON gömülür ve
//     protokolOnEki ile işaretlenir; bu sayede normal sohbet/DM
//     mesajlarından ayırt edilir (AND ve Özel Chat onları göstermez).
//     Grup durumunu değiştiren mesajlar (davet ve yanıtı, üye listesi,
//     ayrılma, kapatma) giden kutusuyla gider: alıcı çevrimdışıysa AND
//     bekletir, bağlandığında iletir. Alıcı tarafta DeltaX kapalıyken
//     gelenler de kaybolmaz (bkz. gecmis.go).
//   - Dosya aktarımı (AND'ın /api/v1/file/*) → asıl veri: proje
//     nesnelerinin (commit/ağaç/blob) grup sırrıyla AES-256-GCM ile
//     şifrelenmiş tar.gz paketi. AND'ın var olan onay akışı (consent)
//     korunur; yalnızca gönderenin peer ID'si bilinen bir grup üyesiyse
//     otomatik kabul edilir (bkz. main.go: consentPollResult işleyicisi).
//
// Bu tasarım gereği veriler AND'ın forum/sohbet pubsub kanallarına hiç
// değmez — yalnızca doğrudan (nokta-noktaya) libp2p akışları üzerinden,
// yalnızca ilgili grubun üyeleri arasında dolaşır: "merkezi ağ dışında
// özel bir ağ".

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lucian95511/and/protokol"
)

// reRastgeleID, rastgeleID()'nin ürettiği biçimi (16 küçük harf hex
// karakteri) doğrular. paketAyristir bunu grupID/projeID'yi bir dosya
// adından çıkarırken kullanır — bunlar filepath.Join ile doğrudan disk
// yollarına eklendiğinden (bkz. projeKokDizini/projeCalismaDizini/
// projeNesneDizini), bu biçim kontrolü olmadan "../../../tmp/evil" gibi bir
// dosya adı, çalışma dizini dışına dizin/dosya oluşturulmasına (path
// traversal) yol açabilir.
var reRastgeleID = regexp.MustCompile(`^[0-9a-f]{16}$`)

const protokolOnEki = protokol.DeltaXOnEki + "1|"

type mesajTipi string

const (
	mDavet       mesajTipi = "davet"
	mUyeGuncelle mesajTipi = "uye_guncelle"
	mDavetYanit  mesajTipi = "davet_yanit"
	mPushBildir  mesajTipi = "push_bildir"
	mPullIstek   mesajTipi = "pull_istek"
	mAyril       mesajTipi = "ayril"
	mGrupKapat   mesajTipi = "grup_kapat"
)

type kontrolZarfi struct {
	Tip     mesajTipi       `json:"tip"`
	GrupID  string          `json:"grup_id,omitempty"`
	ProjeID string          `json:"proje_id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type davetPayload struct {
	GrupAd      string `json:"grup_ad"`
	Sir         string `json:"sir"`
	Kurucu      Uye    `json:"kurucu"`                // her zaman gerçek kurucu — değişmez
	Yoneticiler []Uye  `json:"yoneticiler,omitempty"` // kurucunun yetkilendirdiği yedek kurucular
	Uyeler      []Uye  `json:"uyeler"`
	// Gonderen, bu mesajı gönderen yetkilidir (kurucu ya da bir yönetici).
	// Alıcı, bunu zaten bildiği KurucuPeer ya da Yoneticiler listesiyle
	// karşılaştırıp doğrular — bkz. model.go: zarfIsle.
	Gonderen Uye `json:"gonderen"`
}

type davetYanitPayload struct {
	Kabul bool `json:"kabul"`
	Uye   Uye  `json:"uye"`
}

type pushBildirPayload struct {
	Dal         string `json:"dal"`
	HeadHash    string `json:"head_hash"`
	NesneSayisi int    `json:"nesne_sayisi"`
	DosyaAdi    string `json:"dosya_adi"`
}

type pullIstekPayload struct {
	Dal         string `json:"dal"`
	BilinenHash string `json:"bilinen_hash"`
	Isteyen     Uye    `json:"isteyen"`
}

type ayrilPayload struct {
	Uye Uye `json:"uye"`
}

// grupKapatPayload, mGrupKapat mesajının gerçekten yetkili biri (kurucu ya
// da bir yönetici) tarafından gönderildiğini doğrulamak için kullanılır —
// davet/roster güncellemelerinde uygulanan aynı "yalnızca zaten bildiğimiz
// bir yetkiliden kabul et" ilkesi burada da geçerlidir.
type grupKapatPayload struct {
	Gonderen Uye `json:"gonderen"`
}

func zarfKodla(z kontrolZarfi) (string, error) {
	data, err := json.Marshal(z)
	if err != nil {
		return "", err
	}
	return protokolOnEki + base64.StdEncoding.EncodeToString(data), nil
}

func zarfCoz(metin string) (kontrolZarfi, bool) {
	if !strings.HasPrefix(metin, protokolOnEki) {
		return kontrolZarfi{}, false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(metin, protokolOnEki))
	if err != nil {
		return kontrolZarfi{}, false
	}
	var z kontrolZarfi
	if err := json.Unmarshal(raw, &z); err != nil {
		return kontrolZarfi{}, false
	}
	return z, true
}

// ─── Davetler ───────────────────────────────────────────────────────────────

func davetPayloadOlustur(g Grup, gonderen Uye) davetPayload {
	return davetPayload{
		GrupAd:      g.Ad,
		Sir:         g.Sir,
		Kurucu:      Uye{Ad: g.KurucuAd, PeerID: g.KurucuPeer},
		Yoneticiler: g.Yoneticiler,
		Uyeler:      g.Uyeler,
		Gonderen:    gonderen,
	}
}

// davetGonder, kendi (kurucu ya da bir yönetici) adına hedefPeer'a davet
// gönderir.
func davetGonder(c istemci, hedefPeer string, g Grup, kendi Uye) error {
	payload, err := json.Marshal(davetPayloadOlustur(g, kendi))
	if err != nil {
		return err
	}
	z := kontrolZarfi{Tip: mDavet, GrupID: g.ID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return err
	}
	return c.SendDMQueued(hedefPeer, metin)
}

func davetYanitGonder(c istemci, hedefPeer, grupID string, kabul bool, kendi Uye) error {
	payload, err := json.Marshal(davetYanitPayload{Kabul: kabul, Uye: kendi})
	if err != nil {
		return err
	}
	z := kontrolZarfi{Tip: mDavetYanit, GrupID: grupID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return err
	}
	return c.SendDMQueued(hedefPeer, metin)
}

// uyeGuncellemesiYayinla, güncel üye/yönetici listesini gruptaki herkese
// (kendisi hariç) yeniden gönderir — böylece üyelik değişiklikleri tüm
// üyelerde yakınsar. Kurucu ya da bir yönetici çağırabilir.
func uyeGuncellemesiYayinla(c istemci, g Grup, kendi Uye) []error {
	payload, err := json.Marshal(davetPayloadOlustur(g, kendi))
	if err != nil {
		return []error{err}
	}
	z := kontrolZarfi{Tip: mUyeGuncelle, GrupID: g.ID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return []error{err}
	}
	return herkeseGonder(c, g.Uyeler, kendi.PeerID, metin)
}

// ayrilBildir, bir üyenin gruptan ayrıldığını kurucuya VE tüm yöneticilere
// bildirir (kurucu ulaşılamaz durumdaysa bir yönetici üyeyi listeden
// çıkarabilsin diye).
func ayrilBildir(c istemci, g Grup, kendi Uye) []error {
	payload, err := json.Marshal(ayrilPayload{Uye: kendi})
	if err != nil {
		return []error{err}
	}
	z := kontrolZarfi{Tip: mAyril, GrupID: g.ID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return []error{err}
	}
	hedefler := append([]Uye{{PeerID: g.KurucuPeer, Ad: g.KurucuAd}}, g.Yoneticiler...)
	return herkeseGonder(c, hedefler, kendi.PeerID, metin)
}

// grupKapatBildir, grubun kapatıldığını (dağıtıldığını) gruptaki herkese
// (kendisi hariç) bildirir. Kurucu ya da bir yönetici çağırabilir.
func grupKapatBildir(c istemci, g Grup, kendi Uye) []error {
	payload, err := json.Marshal(grupKapatPayload{Gonderen: kendi})
	if err != nil {
		return []error{err}
	}
	z := kontrolZarfi{Tip: mGrupKapat, GrupID: g.ID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return []error{err}
	}
	return herkeseGonder(c, g.Uyeler, kendi.PeerID, metin)
}

// herkeseGonder, metni hedeflere (kendisi ve tekrarlar hariç) giden kutusu
// üzerinden gönderir: çevrimdışı olan, bağlandığında alır. Gönderimler
// paraleldir; ulaşılamayan bir üyeye doğrudan gönderim denemesi (AND'da
// sıraya almadan önce birkaç saniye sürer) ötekileri bekletmez.
func herkeseGonder(c istemci, hedefler []Uye, kendiPeer, metin string) []error {
	var (
		mu      sync.Mutex
		hatalar []error
		wg      sync.WaitGroup
	)
	gorulen := map[string]bool{"": true, kendiPeer: true}
	for _, h := range hedefler {
		if gorulen[h.PeerID] {
			continue
		}
		gorulen[h.PeerID] = true
		wg.Go(func() {
			if err := c.SendDMQueued(h.PeerID, metin); err != nil {
				mu.Lock()
				hatalar = append(hatalar, fmt.Errorf("%s: %w", h.Ad, err))
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return hatalar
}

// ─── Paket oluşturma / uygulama ─────────────────────────────────────────────

type paketManifest struct {
	Dal      string `json:"dal"`
	HeadHash string `json:"head"`
}

const manifestGirdiAdi = "MANIFEST"

// paketOlustur, kokCommit'ten ulaşılabilen ve bilinenNesneler kümesinde
// olmayan tüm commit/ağaç/blob nesnelerini toplar ve gzip'li bir tar
// arşivi olarak döndürür.
func paketOlustur(dataDir, projeID, dal, kokCommit string, bilinenNesneler map[string]bool) ([]byte, int, error) {
	toplanan := map[string]bool{}
	var sirali []string
	kuyruk := []string{kokCommit}
	for len(kuyruk) > 0 {
		h := kuyruk[0]
		kuyruk = kuyruk[1:]
		if h == "" || toplanan[h] || bilinenNesneler[h] {
			continue
		}
		toplanan[h] = true
		sirali = append(sirali, h)
		tip, payload, err := nesneOku(dataDir, projeID, h)
		if err != nil {
			// h, kokCommit'ten ulaşılabilir görünen bir nesne (bir
			// commit/ağaç'ın işaret ettiği bir hash), yani yerel depoda
			// bulunması gerekirdi. Sessizce atlamak (eskiden olduğu gibi)
			// bu nesnenin çocuklarının (bir ağaç/blob'un referans ettiği
			// hash'lerin) de asla kuyruğa eklenmemesi anlamına gelirdi —
			// gönderilen paket, alıcı tarafın gerçekte eksik olduğunu asla
			// bilemeyeceği şekilde SESSİZCE eksik/kırık olurdu. Yerel depo
			// bozulmuşsa bunu push'u başarısız kılan bir hata olarak
			// bildirmek, "başarılı" görünen ama tamamlanamayan bir
			// senkronizasyondan çok daha güvenlidir.
			return nil, 0, fmt.Errorf("nesne %s okunamadı — yerel depo bozuk olabilir: %w", h, err)
		}
		switch tip {
		case 'C':
			var c Commit
			if json.Unmarshal(payload, &c) == nil {
				kuyruk = append(kuyruk, c.Agac)
				kuyruk = append(kuyruk, c.Ebeveynler...)
			}
		case 'T':
			var t Agac
			if json.Unmarshal(payload, &t) == nil {
				for _, girdi := range t.Girdiler {
					kuyruk = append(kuyruk, girdi.Hash)
				}
			}
		}
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	manifest, err := json.Marshal(paketManifest{Dal: dal, HeadHash: kokCommit})
	if err != nil {
		return nil, 0, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: manifestGirdiAdi, Size: int64(len(manifest)), Mode: 0o600, ModTime: time.Now()}); err != nil {
		return nil, 0, err
	}
	if _, err := tw.Write(manifest); err != nil {
		return nil, 0, err
	}

	for _, h := range sirali {
		path, err := nesneYolu(dataDir, projeID, h)
		if err != nil {
			return nil, 0, fmt.Errorf("nesne %s için yol hesaplanamadı: %w", h, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			// Az önce (yukarıdaki taramada) başarıyla okunmuştu — burada
			// başarısız olması bir yarış (paralel silme) ya da diskte
			// sonradan oluşan bir bozulma anlamına gelir. Eskiden bu da
			// sessizce atlanıp sirali/nesneSayisi'nde sayılmaya devam
			// ederdi, oysa nesne pakete hiç yazılmamış olurdu.
			return nil, 0, fmt.Errorf("nesne %s okunamadı: %w", h, err)
		}
		if err := tw.WriteHeader(&tar.Header{Name: h, Size: int64(len(data)), Mode: 0o600, ModTime: time.Now()}); err != nil {
			return nil, 0, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, 0, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, 0, err
	}
	if err := gz.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(sirali), nil
}

// paketUygula, şifresi çözülmüş bir paketin içindeki nesneleri doğrulayıp
// yerel nesne deposuna yazar. Her nesnenin hash'i yeniden hesaplanarak
// tar girdisinin adıyla eşleştiği doğrulanır — bütünlük ihlali olan
// girdiler sessizce atlanır.
func paketUygula(dataDir, projeID string, veriler []byte) (paketManifest, int, error) {
	var manifest paketManifest
	gz, err := gzip.NewReader(bytes.NewReader(veriler))
	if err != nil {
		return manifest, 0, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	if err := os.MkdirAll(projeNesneDizini(dataDir, projeID), 0o700); err != nil {
		return manifest, 0, err
	}

	sayac := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return manifest, sayac, err
		}
		// maxDosyaBoyutu (vcs.go) ile aynı sınır kullanılır: agacOlustur bir
		// commit'e bu boyuta kadar dosyaları dahil etmeye izin verir; burada
		// daha küçük, bağımsız bir sabit kullanmak (eskiden 64 MB), yerelde
		// geçerli kabul edilip commit'lenmiş 65-200 MB arası bir dosyanın,
		// alıcı tarafta hiçbir hata/uyarı olmadan sessizce atlanmasına
		// (dangling blob referansı bırakarak) yol açardı.
		if hdr.Size < 0 || hdr.Size > maxDosyaBoyutu {
			continue // tekil nesne için makul olmayan boyut — atla
		}
		data := make([]byte, hdr.Size)
		if _, err := io.ReadFull(tr, data); err != nil {
			return manifest, sayac, err
		}

		if hdr.Name == manifestGirdiAdi {
			_ = json.Unmarshal(data, &manifest)
			continue
		}
		if !reHex64.MatchString(hdr.Name) || len(data) < 1 {
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != hdr.Name {
			continue // bütünlük hatası — atla
		}
		if _, err := nesneYaz(dataDir, projeID, data[0], data[1:]); err != nil {
			return manifest, sayac, err
		}
		sayac++
	}
	return manifest, sayac, nil
}

// ─── Gönder (push) ──────────────────────────────────────────────────────────

func paketDosyaAdi(grupID, projeID string) string {
	return fmt.Sprintf("projeag-%s-%s-%d.pack", grupID, projeID, time.Now().UnixNano())
}

// paketAyristir, "projeag-<grupID>-<projeID>-<ts>.pack" biçimindeki dosya
// adından grup ve proje ID'lerini çıkarır.
func paketAyristir(dosyaAdi string) (grupID, projeID string, ok bool) {
	base := strings.TrimSuffix(filepath.Base(dosyaAdi), ".pack")
	parcalar := strings.Split(base, "-")
	if len(parcalar) != 4 || parcalar[0] != "projeag" {
		return "", "", false
	}
	// grupID/projeID her zaman rastgeleID()'nin ürettiği biçimde olmalı —
	// bkz. reRastgeleID'nin doc yorumu. Bu kontrol olmadan, gönderenin
	// seçtiği dosya adı doğrudan disk yollarına eklenir.
	if !reRastgeleID.MatchString(parcalar[1]) || !reRastgeleID.MatchString(parcalar[2]) {
		return "", "", false
	}
	return parcalar[1], parcalar[2], true
}

// pushYap, aktif daldaki yeni nesneleri hedef üyeye şifreli paket olarak
// gönderir ve ardından bir bildirim DM'i yollar. bilinenBaslangic boşsa
// (hiç senkron olmamışsa) tam paket, doluysa yalnızca o commit'ten sonraki
// fark gönderilir.
func pushYap(c istemci, dataDir string, g Grup, p Proje, hedefUye Uye, bilinenBaslangic string) (int, error) {
	head := p.Dallar[p.AktifDal]
	if head == "" {
		return 0, fmt.Errorf("önce en az bir kayıt (commit) oluşturmalısın")
	}
	bilinen := map[string]bool{}
	if bilinenBaslangic != "" {
		bilinen = atalariTopla(dataDir, p.ID, bilinenBaslangic)
		bilinen[bilinenBaslangic] = true
	}
	paket, nesneSayisi, err := paketOlustur(dataDir, p.ID, p.AktifDal, head, bilinen)
	if err != nil {
		return 0, err
	}
	if nesneSayisi == 0 {
		return 0, fmt.Errorf("%s zaten güncel", hedefUye.Ad)
	}
	sifreli, err := sirliSifrele(g.Sir, paket)
	if err != nil {
		return 0, err
	}

	gecici := gecicilerDizini()
	if err := os.MkdirAll(gecici, 0o700); err != nil {
		return 0, err
	}
	dosyaAdi := paketDosyaAdi(g.ID, p.ID)
	yol := filepath.Join(gecici, dosyaAdi)
	if err := os.WriteFile(yol, sifreli, 0o600); err != nil {
		return 0, err
	}
	defer os.Remove(yol) //nolint:errcheck

	if err := c.SendFile(hedefUye.PeerID, yol); err != nil {
		return 0, err
	}

	bp, err := json.Marshal(pushBildirPayload{Dal: p.AktifDal, HeadHash: head, NesneSayisi: nesneSayisi, DosyaAdi: dosyaAdi})
	if err != nil {
		return nesneSayisi, err
	}
	z := kontrolZarfi{Tip: mPushBildir, GrupID: g.ID, ProjeID: p.ID, Payload: bp}
	metin, err := zarfKodla(z)
	if err != nil {
		return nesneSayisi, err
	}
	return nesneSayisi, c.SendDM(hedefUye.PeerID, metin)
}

// ─── Çek (pull) isteği ──────────────────────────────────────────────────────

func pullIste(c istemci, hedefUye Uye, g Grup, p Proje, kendi Uye) error {
	bilinen := p.Dallar[p.AktifDal]
	payload, err := json.Marshal(pullIstekPayload{Dal: p.AktifDal, BilinenHash: bilinen, Isteyen: kendi})
	if err != nil {
		return err
	}
	z := kontrolZarfi{Tip: mPullIstek, GrupID: g.ID, ProjeID: p.ID, Payload: payload}
	metin, err := zarfKodla(z)
	if err != nil {
		return err
	}
	return c.SendDM(hedefUye.PeerID, metin)
}

// ─── Gelen paket işleme ─────────────────────────────────────────────────────

// paketGonderenGuvenilirMi, gelen bir dosya transferi isteğinin adından
// hedef grubu ayrıştırıp gönderenin O BELİRLİ grubun üyesi olup olmadığını
// kontrol eder — "gönderen herhangi bir grubumun üyesi mi" gibi gevşek bir
// kontrol, gönderenin üyesi olduğu bir grup için, üyesi olmadığı başka bir
// gruba ait paketi göndermesine izin verir (paket zaten pushPaketiIsle'de
// o grubun üyeliğine göre reddedilir, ama gereksiz bir dosya transferini
// baştan, onay ekranı göstermeden geçirmiş olur).
func paketGonderenGuvenilirMi(dataDir, dosyaAdi, gonderenPeerID string) bool {
	grupID, _, ok := paketAyristir(dosyaAdi)
	if !ok {
		return false
	}
	g, err := grupOku(dataDir, grupID)
	if err != nil {
		return false
	}
	return g.uyeMi(gonderenPeerID)
}

// pushPaketiIsle, dosya aktarımıyla gelen bir .pack dosyasını doğrular,
// şifresini çözer ve nesnelerini yerel depoya uygular. gonderenPeerID,
// AND'ın filemgr bileşeninden gelen doğrulanmış (sahte olamayan) libp2p
// eş kimliğidir — güvenlik kontrolü buna dayanır.
func pushPaketiIsle(dataDir, dosyaYolu, gonderenPeerID string) (grupID, projeID string, manifest paketManifest, nesneSayisi int, err error) {
	grupID, projeID, ok := paketAyristir(dosyaYolu)
	if !ok {
		return "", "", manifest, 0, fmt.Errorf("proje ağı paketi değil")
	}
	g, err := grupOku(dataDir, grupID)
	if err != nil {
		return grupID, projeID, manifest, 0, fmt.Errorf("bilinmeyen grup: %w", err)
	}
	if !g.uyeMi(gonderenPeerID) {
		return grupID, projeID, manifest, 0, fmt.Errorf("gönderen %s grubunun üyesi değil", g.Ad)
	}
	sifreli, err := os.ReadFile(dosyaYolu)
	if err != nil {
		return grupID, projeID, manifest, 0, err
	}
	acik, err := sirliCoz(g.Sir, sifreli)
	if err != nil {
		return grupID, projeID, manifest, 0, fmt.Errorf("paket çözülemedi (yanlış grup anahtarı olabilir): %w", err)
	}
	if _, err := os.Stat(projeKokDizini(dataDir, projeID)); os.IsNotExist(err) {
		// Bu düğümde proje hiç yoksa iskeletini oluştur — çalışma dizini
		// dahil, aksi halde Durum/Kaydet/Klasörü Aç bu proje için hiç
		// var olmayan bir dizinle karşılaşıp hata verir.
		if err := os.MkdirAll(projeCalismaDizini(dataDir, projeID), 0o700); err != nil {
			return grupID, projeID, manifest, 0, err
		}
		p := Proje{ID: projeID, GrupID: grupID, Ad: "(alınan proje)", OlusturmaTS: time.Now().UTC(), Dallar: map[string]string{}, AktifDal: "ana"}
		if err := projeKaydet(dataDir, p); err != nil {
			return grupID, projeID, manifest, 0, err
		}
	}
	manifest, nesneSayisi, err = paketUygula(dataDir, projeID, acik)
	return grupID, projeID, manifest, nesneSayisi, err
}
