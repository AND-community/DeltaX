package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/internal/pluginapi"
	"github.com/lucian95511/and/internal/ui"
)

// ─── Ekranlar ──────────────────────────────────────────────────────────────

type ekran int

const (
	ekAnaMenu ekran = iota
	ekKimlik
	ekGrupListesi
	ekGrupOlustur
	ekGrupPaneli
	ekUyeDavet
	ekDavetler
	ekProjeOlustur
	ekProjePaneli
	ekDurum
	ekCommitMesaj
	ekGunluk
	ekDallar
	ekYeniDal
	ekKayit
	ekDiff
	ekGonderSecim
	ekUzakDallar
	ekPullIste
	ekDosyaOnay
	ekOnay
	ekYoneticiSecim
	ekDosyaListesi
	ekYeniDosya
	ekDosyaDuzenle
	ekDisaAktar
	ekYardim
)

// kisayol, ? yardım kutusunda gösterilen tek bir klavye kısayolunu tanımlar.
type kisayol struct {
	tus      string
	aciklama string
}

// onayTipi, genel onay ekranının (ekOnay) hangi yıkıcı işlem için gösterildiğini belirtir.
type onayTipi int

const (
	onayGrupCikKapat onayTipi = iota
	onayDalDegistir
	onayBirlestir
	onayGeriAl // çalışma klasöründeki bir değişikliği geri al (m.geriAlYol)
	onayDalSil // bir dalı sil (m.dallar[m.dalIdx])
)

// bekleyenDavet, kabul edilmemiş bir grup davetidir. store.go'daki
// davetleriKaydet/davetleriYukle ile diske de yazılır — eklenti kapatılıp
// yeniden açılsa (ya da AND yeniden başlasa) bile davet kaybolmaz.
type bekleyenDavet struct {
	GrupID       string       `json:"grup_id"`
	GonderenPeer string       `json:"gonderen_peer"`
	Payload      davetPayload `json:"payload"`
}

// ─── tea.Msg tipleri ─────────────────────────────────────────────────────────

type dmPollResultMsg struct {
	msgs []pluginapi.DMMsg
	err  error
}
type filePollResultMsg struct {
	msgs []pluginapi.FileMsg
	err  error
}
type consentPollResultMsg struct {
	reqs []pluginapi.FileConsentReq
	err  error
}
type peersResultMsg struct {
	peers []pluginapi.PeerInfo
	err   error
}
type basitSonuc struct {
	islem string
	err   error
}
type grupOlusturSonucu struct {
	grup Grup
	err  error
}
type kabulSonucu struct {
	grup Grup
	err  error
}
type projeOlusturSonucu struct {
	proje Proje
	err   error
}
type durumSonucu struct {
	projeID string // bkz. commitSonucu'nun doc yorumu
	durum   DurumSonucu
	err     error
}
type commitSonucu struct {
	// projeID, bu sonucun HANGİ proje için hesaplandığını taşır. Update()
	// bunu m.aktifProje.ID ile karşılaştırır: kullanıcı bu async işlem
	// sürerken (m.pending==true iken gezinme tuşları pending'i kontrol
	// etmediğinden) başka bir projeye geçmişse, sonuç artık m.aktifProje'ye
	// ait değildir — projeID kontrolü olmadan eski sonuç yanlışlıkla
	// ŞİMDİKİ (farklı) projeye yazılıp diske kaydedilirdi.
	projeID string
	hash    string
	atlanan int
	err     error
}
type gunlukSonucu struct {
	girdiler []LogGirdisi
}
type dalDegistirSonucu struct {
	projeID string // bkz. commitSonucu'nun doc yorumu
	dal     string
	kirli   bool
	err     error
}
type diffSonucu struct {
	satirlar []DiffSatiri
	err      error
}
type pushSonucu struct {
	hedefAd     string
	nesneSayisi int
	err         error
}
type birlestirSonucu struct {
	projeID    string // bkz. commitSonucu'nun doc yorumu
	catismalar []string
	yeniCommit string
	uzakHash   string
	err        error
}
type grupCikKapatSonucu struct {
	kapatildi bool // true: kurucu grubu kapattı, false: üye gruptan çıktı
	agHatasi  error
	err       error
}
type dosyaListesiSonucu struct {
	liste []string
	err   error
}
type dosyaAcSonucu struct {
	rel    string
	icerik string
	err    error
}
type dosyaKaydetSonucu struct {
	err error
}
type dosyaSilSonucu struct {
	rel string
	err error
}
type disaAktarSonucu struct {
	yol string
	err error
}

// projeOzeti, proje panelinin üst kartıdır: aktif dalın son kaydı,
// çalışma klasörünün durumu ve her uzak uçta yerelde olmayan kayıt sayısı.
type projeOzeti struct {
	projeID  string
	son      *LogGirdisi
	durum    DurumSonucu
	uzakYeni map[string]int
	hata     string
}

type projeOzetiSonucu struct{ ozet projeOzeti }

// kayitDetayi, Günlük'te açılan bir kaydın ayrıntısıdır.
type kayitDetayi struct {
	hash          string
	commit        Commit
	degisiklikler []DosyaDegisikligi
	idx           int
}

type kayitSonucu struct {
	projeID string
	detay   kayitDetayi
	err     error
}

type geriAlSonucu struct {
	projeID string
	yol     string
	err     error
}

// sonKayit, ana menüdeki "Son kayıtlar" listesinin bir satırıdır.
type sonKayit struct {
	grup, proje string
	LogGirdisi
}

type sonKayitlarSonucu struct{ kayitlar []sonKayit }

// ─── Model ────────────────────────────────────────────────────────────────────

type model struct {
	client   istemci
	identity pluginapi.IdentityInfo
	dataDir  string

	ekran   ekran
	w, h    int
	notice  string
	isErr   bool
	pending bool

	menuIdx int

	gruplar        []Grup
	grupIdx        int
	aktifGrup      *Grup
	grupAdIn       textinput.Model
	grupAramaAktif bool
	grupAramaIn    textinput.Model

	uyeDavetIn textinput.Model

	bekleyenDavetler []bekleyenDavet
	davetIdx         int

	// gonderilenDavetler: mDavetYanit işleyicisi bir "kabul"ü yalnızca
	// gerçekten davet edilmiş bir eşten gelirse uygular (bkz. davetDefteri).
	gonderilenDavetler davetDefteri

	// imlec: DM geçmişinde nereye kadar işlendiği (bkz. gecmis.go).
	imlec dmImleci

	projeler   []Proje
	projeIdx   int
	aktifProje *Proje
	projeAdIn  textinput.Model

	projePanelIdx int

	durum     DurumSonucu
	durumIdx  int
	geriAlYol string

	// ozet: proje panelinin üst kartı (bkz. projeOzetiCmd); nil iken hesaplanıyor.
	ozet *projeOzeti
	// projeSonKayit: grup panelinde her projenin son kaydı (proje ID → kayıt).
	projeSonKayit map[string]LogGirdisi
	// sonKayitlar: ana menüdeki, tüm projelerin en yeni kayıtları.
	sonKayitlar []sonKayit
	// esler: bağlı eşler (davet ekranında AND düğümleri önerilir).
	esler    []pluginapi.PeerInfo
	oneriIdx int

	commitMsgIn textinput.Model

	gunluk           []LogGirdisi
	gunlukIdx        int
	gunlukAramaAktif bool
	gunlukAramaIn    textinput.Model
	kayit            kayitDetayi

	yeniDalIn textinput.Model
	dalIdx    int
	dallar    []string
	// dalSon: Dallar ekranında her dalın son kaydı.
	dalSon map[string]LogGirdisi
	// dalKaynak: yeni dalın başlayacağı kayıt; boşsa aktif dalın son kaydı.
	dalKaynak string

	diffVP      viewport.Model
	diffBaslik  string
	diffGeri    ekran
	diffEklenen int
	diffSilinen int

	pushIdx   int
	pullIdx   int
	uyelerHar []Uye // aktif grubun kendisi hariç üyeleri (push/pull hedef listesi)

	yoneticiHar []Uye
	yoneticiIdx int

	cevrimici map[string]bool

	uzakAnahtarlari []string
	uzakIdx         int

	pendingConsent *pluginapi.FileConsentReq
	onayOnceki     ekran

	sonCatismalar []string

	dosyaListesi    []string
	dosyaIdx        int
	dosyaSilOnayi   string // 'd' ikinci kez basılana kadar silme onayı bekleyen dosya
	dosyaDuzenleYol string
	dosyaDuzenleTA  textarea.Model
	yeniDosyaIn     textinput.Model

	disaAktarIn textinput.Model

	spinner spinner.Model

	yardimGeriEkran ekran

	onayTip       onayTipi
	onayGeriEkran ekran
}

func newModel(c istemci, id pluginapi.IdentityInfo, dataDir string) model {
	grupAdIn := textinput.New()
	grupAdIn.Placeholder = "Grup adı…"
	grupAdIn.CharLimit = 80

	grupAramaIn := textinput.New()
	grupAramaIn.Placeholder = "Ara (grup/kurucu adı)…"
	grupAramaIn.CharLimit = 100

	uyeDavetIn := textinput.New()
	uyeDavetIn.Placeholder = "Davet edilecek kişinin peer ID'si (12D3Koo…)"
	uyeDavetIn.CharLimit = 128

	projeAdIn := textinput.New()
	projeAdIn.Placeholder = "Proje adı…"
	projeAdIn.CharLimit = 80

	commitMsgIn := textinput.New()
	commitMsgIn.Placeholder = "Kayıt (commit) mesajı…"
	commitMsgIn.CharLimit = 200

	yeniDalIn := textinput.New()
	yeniDalIn.Placeholder = "Yeni dal adı…"
	yeniDalIn.CharLimit = 60

	yeniDosyaIn := textinput.New()
	yeniDosyaIn.Placeholder = "Göreli dosya yolu (ör. src/ana.go)…"
	yeniDosyaIn.CharLimit = 240

	gunlukAramaIn := textinput.New()
	gunlukAramaIn.Placeholder = "Ara (mesaj/yazar/hash)…"
	gunlukAramaIn.CharLimit = 100

	disaAktarIn := textinput.New()
	disaAktarIn.Placeholder = "Hedef klasör…"
	disaAktarIn.CharLimit = 300
	disaAktarIn.SetValue(varsayilanDisaAktarKlasoru())

	dosyaDuzenleTA := textarea.New()
	dosyaDuzenleTA.ShowLineNumbers = true

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	// Giriş kutuları ve düzenleyici AND'ın etkin temasıyla çizilir.
	for _, in := range []*textinput.Model{&grupAdIn, &grupAramaIn, &uyeDavetIn, &projeAdIn, &commitMsgIn,
		&yeniDalIn, &yeniDosyaIn, &gunlukAramaIn, &disaAktarIn} {
		ui.StyleInput(in)
	}
	ui.StyleTextarea(&dosyaDuzenleTA)

	return model{
		client:             c,
		identity:           id,
		dataDir:            dataDir,
		ekran:              ekAnaMenu,
		grupAdIn:           grupAdIn,
		grupAramaIn:        grupAramaIn,
		uyeDavetIn:         uyeDavetIn,
		projeAdIn:          projeAdIn,
		commitMsgIn:        commitMsgIn,
		yeniDalIn:          yeniDalIn,
		yeniDosyaIn:        yeniDosyaIn,
		dosyaDuzenleTA:     dosyaDuzenleTA,
		gunlukAramaIn:      gunlukAramaIn,
		disaAktarIn:        disaAktarIn,
		cevrimici:          map[string]bool{},
		spinner:            sp,
		gruplar:            gruplariListele(dataDir),
		bekleyenDavetler:   davetleriYukle(dataDir),
		gonderilenDavetler: davetDefteriniOku(dataDir, time.Now()),
		imlec:              imleciOku(dataDir),
	}
}

func (m model) kendiUye() Uye {
	return Uye{Ad: m.identity.Name, PeerID: m.identity.PeerID, PubKey: m.identity.PubKey}
}

func (m model) calismaDir() string {
	return projeCalismaDizini(m.dataDir, m.aktifProje.ID)
}

// projeSonucuGuncelMi, projeID (bir arka plan komutunu başlatırken sabitlenen
// proje) hâlâ m.aktifProje ile eşleşiyor mu diye bakar. Gezinme tuşları
// (proje/grup listesine dönüp başka bir proje seçmek gibi) m.pending'i
// kontrol etmediğinden, bir işlem sürerken kullanıcı başka bir projeye
// geçebilir — bu durumda gelen sonuç artık m.aktifProje'ye ait değildir ve
// commitSonucu/dalDegistirSonucu/birlestirSonucu gibi diske yazan
// işleyicilerin bunu uygulaması, o an görüntülenen (farklı) projenin
// dosyasını yanlış veriyle ezerdi.
func (m model) projeSonucuGuncelMi(projeID string) bool {
	return m.aktifProje != nil && m.aktifProje.ID == projeID
}

// bildirGuncelle, arka plandan (zarfIsle, filePollResultMsg) gelen bir
// BİLGİLENDİRME mesajının, kullanıcının henüz görmediği bir HATA mesajının
// üzerine sessizce yazılmasını engeller — sürekli dönen DM/dosya yoklaması bir
// hatayı okumaya fırsat bulamadan değiştirmesin diye. Yeni mesaj kendisi bir
// hataysa (hata=true) her zaman gösterilir; iki hata arasında en yenisi kazanır.
func (m model) bildirGuncelle(mesaj string, hata bool) model {
	if m.isErr && !hata {
		return m
	}
	m.notice, m.isErr = mesaj, hata
	return m
}

// kisayollarIcin, verilen ekranda geçerli olan klavye kısayollarını döndürür
// — ? yardım kutusunun (bkz. Update/viewYardim) tek veri kaynağıdır. Yalnızca
// metin girişi OLMAYAN liste/menü ekranları kapsanır: bir ekran burada yoksa
// (nil döner) o ekranda ? sıradan bir tuş olarak metne yazılır/yok sayılır ve
// yardım kutusu hiç açılmaz.
func (m model) kisayollarIcin(e ekran) []kisayol {
	switch e {
	case ekAnaMenu:
		return []kisayol{{"↑/↓, j/k", "menüde gez"}, {"enter", "seç"}, {"q, esc", "çıkış"}}
	case ekKimlik:
		return []kisayol{{"esc", "geri"}}
	case ekGrupListesi:
		return []kisayol{{"↑/↓", "grup seç"}, {"enter", "grubu aç"}, {"/", "ara"}, {"r", "yenile"}, {"esc", "geri"}}
	case ekGrupPaneli:
		ks := []kisayol{{"↑/↓", "proje seç"}, {"enter", "projeyi aç"}, {"n", "yeni proje"}}
		if m.aktifGrup != nil {
			if m.aktifGrup.yetkiliMi(m.identity.PeerID) {
				ks = append(ks, kisayol{"i", "üye davet et"})
			}
			if m.aktifGrup.benKurucuMuyum(m.identity.PeerID) {
				ks = append(ks, kisayol{"y", "yönetici ata/kaldır"})
			}
			if m.aktifGrup.yetkiliMi(m.identity.PeerID) {
				ks = append(ks, kisayol{"x", "grubu kapat"})
			} else {
				ks = append(ks, kisayol{"x", "gruptan çık"})
			}
		}
		return append(ks, kisayol{"r", "yenile"}, kisayol{"esc", "geri"})
	case ekDavetler:
		return []kisayol{{"↑/↓", "davet seç"}, {"a", "kabul et"}, {"r", "reddet"}, {"esc", "geri"}}
	case ekProjePaneli:
		ks := []kisayol{{"↑/↓", "işlem seç"}, {"enter", "çalıştır"}}
		for _, is := range projeIslemleri {
			ks = append(ks, kisayol{is.tus, is.ad})
		}
		return append(ks, kisayol{"esc", "geri"})
	case ekDurum:
		return []kisayol{{"↑/↓", "dosya seç"}, {"enter", "farkı göster"}, {"g", "değişikliği geri al"}, {"c", "kaydet (commit)"}, {"r", "yenile"}, {"esc", "geri"}}
	case ekGunluk:
		if m.gunlukAramaAktif {
			return nil
		}
		return []kisayol{{"↑/↓", "kayıt seç"}, {"enter", "kaydın ayrıntısı"}, {"/", "ara"}, {"esc", "geri"}}
	case ekKayit:
		return []kisayol{{"↑/↓", "dosya seç"}, {"enter", "dosyanın bu kayıttaki farkı"}, {"b", "bu kayıttan yeni dal"}, {"esc", "günlüğe dön"}}
	case ekDallar:
		return []kisayol{{"↑/↓", "dal seç"}, {"enter", "dala geç"}, {"n", "yeni dal"}, {"d", "dalı sil"}, {"esc", "geri"}}
	case ekDiff:
		return []kisayol{{"↑/↓, pgup/pgdn", "kaydır"}, {"esc", "geri"}}
	case ekGonderSecim:
		return []kisayol{{"↑/↓", "üye seç"}, {"enter", "gönder"}, {"a", "çevrimiçi herkese gönder"}, {"esc", "geri"}}
	case ekUzakDallar:
		return []kisayol{{"↑/↓", "uç seç"}, {"enter", "birleştir"}, {"esc", "geri"}}
	case ekPullIste:
		return []kisayol{{"↑/↓", "üye seç"}, {"enter", "çek isteği gönder"}, {"a", "çevrimiçi herkesten iste"}, {"esc", "geri"}}
	case ekYoneticiSecim:
		return []kisayol{{"↑/↓", "üye seç"}, {"enter", "yönetici yap/kaldır"}, {"esc", "geri"}}
	case ekDosyaListesi:
		return []kisayol{{"↑/↓", "dosya seç"}, {"enter", "düzenle"}, {"n", "yeni dosya"}, {"d", "sil (iki kez bas)"}, {"esc", "geri"}}
	}
	return nil
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		gecmisCekCmd(m.client, m.imlec.ID),
		filePollCmd(m.client),
		consentPollCmd(m.client),
		peersPollCmd(m.client),
		sonKayitlarCmd(m.dataDir),
	)
}

// projeIslemi, proje panelindeki bir işlemdir; tus, listede seçmeden doğrudan
// çalıştıran kısayoldur.
type projeIslemi struct {
	tus, ad, aciklama string
}

var projeIslemleri = []projeIslemi{
	{"d", "Değişiklikler", "kaydedilmemiş değişiklikler, farklar, geri alma"},
	{"c", "Kaydet (Commit)", "çalışma klasörünün anlık görüntüsünü al"},
	{"l", "Günlük", "kayıt geçmişi ve her kaydın ayrıntısı"},
	{"b", "Dallar", "dal oluştur, geç, sil"},
	{"p", "Gönder (Push)", "yeni kayıtları üyelere gönder"},
	{"m", "Uzak Dallar (Birleştir)", "üyelerden gelen değişiklikleri birleştir"},
	{"u", "Üyeden Çek İste (Pull)", "bir üyeden değişikliklerini iste"},
	{"f", "Dosyalar (Düzenle)", "AND'dan çıkmadan dosya düzenle"},
	{"o", "Çalışma Klasörünü Aç", "dosya yöneticisinde aç"},
	{"e", "Dışa Aktar (.zip)", "tüm geçmişi tek dosyaya paketle"},
}

// projeOzetiCmd, proje panelinin üst kartını hesaplar: son kayıt, çalışma
// klasörünün durumu ve her uzak uçta yerelde olmayan kayıt sayısı.
func projeOzetiCmd(dataDir string, p Proje, calismaDir string) tea.Cmd {
	head := p.Dallar[p.AktifDal]
	uzaklar := make(map[string]string, len(p.Uzaklar))
	for k, v := range p.Uzaklar {
		uzaklar[k] = v
	}
	return func() tea.Msg {
		oz := projeOzeti{projeID: p.ID, uzakYeni: map[string]int{}}
		headTree, err := headTreeOf(dataDir, p.ID, head)
		if err != nil {
			oz.hata = describeErr(err)
			return projeOzetiSonucu{oz}
		}
		if head != "" {
			if c, err := commitOku(dataDir, p.ID, head); err == nil {
				oz.son = &LogGirdisi{Hash: head, Commit: c}
			}
		}
		if oz.durum, err = durumHesapla(dataDir, p.ID, calismaDir, headTree); err != nil {
			oz.hata = describeErr(err)
		}
		yerel := atalariTopla(dataDir, p.ID, head)
		for k, h := range uzaklar {
			oz.uzakYeni[k] = yeniKayitSay(dataDir, p.ID, yerel, h)
		}
		return projeOzetiSonucu{oz}
	}
}

// sonKayitlarCmd, tüm grupların tüm projelerinden en yeni kayıtları toplar
// (ana menüdeki "Son kayıtlar").
func sonKayitlarCmd(dataDir string) tea.Cmd {
	return func() tea.Msg {
		var out []sonKayit
		for _, g := range gruplariListele(dataDir) {
			for _, p := range projeleriListele(dataDir, g.ID) {
				for _, k := range gunlukAl(dataDir, p.ID, p.Dallar[p.AktifDal], 3) {
					out = append(out, sonKayit{grup: g.Ad, proje: p.Ad, LogGirdisi: k})
				}
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Commit.TS.After(out[j].Commit.TS) })
		if len(out) > 6 {
			out = out[:6]
		}
		return sonKayitlarSonucu{kayitlar: out}
	}
}

// sonKayitlari, projelerin aktif dallarının son kayıtlarıdır (grup paneli).
func sonKayitlari(dataDir string, projeler []Proje) map[string]LogGirdisi {
	out := map[string]LogGirdisi{}
	for _, p := range projeler {
		if k := gunlukAl(dataDir, p.ID, p.Dallar[p.AktifDal], 1); len(k) > 0 {
			out[p.ID] = k[0]
		}
	}
	return out
}

func kayitCmd(dataDir, projeID, hash string) tea.Cmd {
	return func() tea.Msg {
		c, dd, err := kayitDegisiklikleri(dataDir, projeID, hash)
		return kayitSonucu{projeID: projeID, detay: kayitDetayi{hash: hash, commit: c, degisiklikler: dd}, err: err}
	}
}

func kayitDiffCmd(dataDir, projeID, hash, yol string) tea.Cmd {
	return func() tea.Msg {
		satirlar, err := kayitDosyaFarki(dataDir, projeID, hash, yol)
		return diffSonucu{satirlar: satirlar, err: err}
	}
}

func geriAlCmd(dataDir, projeID, calismaDir, head, yol string) tea.Cmd {
	return func() tea.Msg {
		headTree, err := headTreeOf(dataDir, projeID, head)
		if err == nil {
			err = degisikligiGeriAl(dataDir, projeID, calismaDir, headTree, yol)
		}
		return geriAlSonucu{projeID: projeID, yol: yol, err: err}
	}
}

// ─── Async komutlar ───────────────────────────────────────────────────────────

func dmPollCmd(c istemci) tea.Cmd {
	return func() tea.Msg {
		msgs, err := c.PollDM()
		return dmPollResultMsg{msgs: msgs, err: err}
	}
}

func dmPollCmdAfter(c istemci) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		msgs, err := c.PollDM()
		return dmPollResultMsg{msgs: msgs, err: err}
	}
}

func filePollCmd(c istemci) tea.Cmd {
	return func() tea.Msg {
		msgs, err := c.PollFile()
		return filePollResultMsg{msgs: msgs, err: err}
	}
}

func filePollCmdAfter(c istemci) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		msgs, err := c.PollFile()
		return filePollResultMsg{msgs: msgs, err: err}
	}
}

func consentPollCmd(c istemci) tea.Cmd {
	return func() tea.Msg {
		reqs, err := c.PollConsent()
		return consentPollResultMsg{reqs: reqs, err: err}
	}
}

func peersPollCmd(c istemci) tea.Cmd {
	return func() tea.Msg {
		peers, err := c.Peers()
		return peersResultMsg{peers: peers, err: err}
	}
}

func peersPollCmdAfter(c istemci) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(5 * time.Second)
		peers, err := c.Peers()
		return peersResultMsg{peers: peers, err: err}
	}
}

func onayCmd(c istemci, transferID string, accept bool) tea.Cmd {
	return func() tea.Msg {
		return basitSonuc{islem: "onay", err: c.RespondConsent(transferID, accept)}
	}
}

func grupOlusturCmd(dataDir, ad string, kendi Uye) tea.Cmd {
	return func() tea.Msg {
		sir, err := sirUret()
		if err != nil {
			return grupOlusturSonucu{err: err}
		}
		g := Grup{
			ID: rastgeleID(), Ad: ad, Sir: sir,
			KurucuAd: kendi.Ad, KurucuPeer: kendi.PeerID,
			Uyeler: []Uye{kendi}, OlusturmaTS: time.Now().UTC(),
		}
		if err := grupKaydet(dataDir, g); err != nil {
			return grupOlusturSonucu{err: err}
		}
		return grupOlusturSonucu{grup: g}
	}
}

func davetGonderCmd(c istemci, hedefPeer string, g Grup, kendi Uye) tea.Cmd {
	return func() tea.Msg {
		return basitSonuc{islem: "davet", err: davetGonder(c, hedefPeer, g, kendi)}
	}
}

func kabulCmd(c istemci, dataDir string, davet bekleyenDavet, kendi Uye, kabul bool) tea.Cmd {
	return func() tea.Msg {
		if !kabul {
			return kabulSonucu{err: davetYanitGonder(c, davet.GonderenPeer, davet.GrupID, false, kendi)}
		}
		g := Grup{
			ID: davet.GrupID, Ad: davet.Payload.GrupAd, Sir: davet.Payload.Sir,
			KurucuAd: davet.Payload.Kurucu.Ad, KurucuPeer: davet.Payload.Kurucu.PeerID,
			Yoneticiler: davet.Payload.Yoneticiler,
			Uyeler:      davet.Payload.Uyeler, OlusturmaTS: time.Now().UTC(),
		}
		g.uyeEkleGuncelle(kendi)
		if err := grupKaydet(dataDir, g); err != nil {
			return kabulSonucu{err: err}
		}
		err := davetYanitGonder(c, davet.GonderenPeer, davet.GrupID, true, kendi)
		return kabulSonucu{grup: g, err: err}
	}
}

func rosterYayinCmd(c istemci, g Grup, kendi Uye) tea.Cmd {
	return func() tea.Msg {
		if hatalar := uyeGuncellemesiYayinla(c, g, kendi); len(hatalar) > 0 {
			return basitSonuc{islem: "roster", err: fmt.Errorf("%d üyeye ulaşılamadı", len(hatalar))}
		}
		return basitSonuc{islem: "roster"}
	}
}

// grupTerkEtCmd, gruptan ayrılma ya da (kurucu/yönetici durumunda) grubu
// tamamen kapatma işlemini yürütür: önce ilgili taraflar ağ üzerinden
// bilgilendirilir (en iyi çaba — ulaşılamayanlar agHatasi olarak
// raporlanır), ardından grubun yerel kaydı silinir. Grubun proje verileri
// (nesneler, çalışma klasörü) kasıtlı olarak diskte bırakılır.
func grupTerkEtCmd(c istemci, dataDir string, g Grup, kendi Uye) tea.Cmd {
	return func() tea.Msg {
		kapatildi := g.yetkiliMi(kendi.PeerID)
		var agHatasi error
		if kapatildi {
			if hatalar := grupKapatBildir(c, g, kendi); len(hatalar) > 0 {
				agHatasi = fmt.Errorf("%d üyeye ulaşılamadı", len(hatalar))
			}
		} else if hatalar := ayrilBildir(c, g, kendi); len(hatalar) > 0 {
			agHatasi = fmt.Errorf("%d yetkiliye ulaşılamadı", len(hatalar))
		}
		if err := grupSil(dataDir, g.ID); err != nil {
			return grupCikKapatSonucu{kapatildi: kapatildi, err: err}
		}
		return grupCikKapatSonucu{kapatildi: kapatildi, agHatasi: agHatasi}
	}
}

func projeOlusturCmd(dataDir, grupID, ad string) tea.Cmd {
	return func() tea.Msg {
		p, err := projeOlustur(dataDir, grupID, ad)
		return projeOlusturSonucu{proje: p, err: err}
	}
}

func durumCmd(dataDir, projeID, calismaDir, head string) tea.Cmd {
	return func() tea.Msg {
		headTree, err := headTreeOf(dataDir, projeID, head)
		if err != nil {
			return durumSonucu{projeID: projeID, err: err}
		}
		d, err := durumHesapla(dataDir, projeID, calismaDir, headTree)
		return durumSonucu{projeID: projeID, durum: d, err: err}
	}
}

// headTreeOf, head commit'inin ağaç hash'ini döndürür. head=="" GEÇERLİ bir
// durumdur (henüz hiç kayıt yok / ortak ata bulunamadı — boş ağaç). Ancak
// head boş DEĞİLKEN commitOku başarısız olursa (ör. bozuk/eksik nesne
// dosyası), bu ayrı bir durumdur ve eskiden ikisi de aynı şekilde ""
// döndürülüyordu — çağıran taraf bunu "boş ağaç" sanıp örneğin dalaGec'i
// çalışma dizinini tamamen SİLİP hiçbir şey geri yazmadan (sessiz veri
// kaybı) çağırabiliyordu. Şimdi ikinci durum bir hata olarak ayrıştırılıyor.
func headTreeOf(dataDir, projeID, head string) (string, error) {
	if head == "" {
		return "", nil
	}
	c, err := commitOku(dataDir, projeID, head)
	if err != nil {
		return "", fmt.Errorf("commit %s okunamadı: %w", kisaHash(head), err)
	}
	return c.Agac, nil
}

func commitCmd(dataDir, projeID, calismaDir, head, yazar, mesaj string) tea.Cmd {
	return func() tea.Msg {
		yeniTree, _, atlanan, err := agacOlustur(dataDir, projeID, calismaDir)
		if err != nil {
			return commitSonucu{projeID: projeID, err: err}
		}
		var ebeveynler []string
		if head != "" {
			ebeveynler = []string{head}
		}
		if ikinci, birlesmeVar := birlesmeBekleniyorMu(dataDir, projeID); birlesmeVar {
			ebeveynler = append(ebeveynler, ikinci)
		} else if head != "" {
			if hc, err := commitOku(dataDir, projeID, head); err == nil && hc.Agac == yeniTree {
				return commitSonucu{projeID: projeID, err: fmt.Errorf("kaydedilecek değişiklik yok")}
			}
		}
		hash, err := commitOlustur(dataDir, projeID, yeniTree, ebeveynler, yazar, mesaj)
		if err != nil {
			return commitSonucu{projeID: projeID, err: err}
		}
		birlesmeTemizle(dataDir, projeID)
		return commitSonucu{projeID: projeID, hash: hash, atlanan: len(atlanan)}
	}
}

func gunlukCmd(dataDir, projeID, head string) tea.Cmd {
	return func() tea.Msg {
		return gunlukSonucu{girdiler: gunlukAl(dataDir, projeID, head, 200)}
	}
}

func dalDegistirCmd(dataDir, projeID, calismaDir, mevcutHead, hedefDal, hedefHead string) tea.Cmd {
	return func() tea.Msg {
		mevcutTree, err := headTreeOf(dataDir, projeID, mevcutHead)
		if err != nil {
			return dalDegistirSonucu{projeID: projeID, err: err}
		}
		durum, err := durumHesapla(dataDir, projeID, calismaDir, mevcutTree)
		if err != nil {
			return dalDegistirSonucu{projeID: projeID, err: err}
		}
		if !durum.temizMi() {
			return dalDegistirSonucu{projeID: projeID, kirli: true}
		}
		hedefTree, err := headTreeOf(dataDir, projeID, hedefHead)
		if err != nil {
			return dalDegistirSonucu{projeID: projeID, err: err}
		}
		if err := dalaGec(dataDir, projeID, calismaDir, hedefTree); err != nil {
			return dalDegistirSonucu{projeID: projeID, err: err}
		}
		return dalDegistirSonucu{projeID: projeID, dal: hedefDal}
	}
}

func diffCmd(dataDir, projeID, calismaDir, head, yol string) tea.Cmd {
	return func() tea.Msg {
		headTree, err := headTreeOf(dataDir, projeID, head)
		if err != nil {
			return diffSonucu{err: err}
		}
		satirlar, err := dosyaFarkiHesapla(dataDir, projeID, calismaDir, headTree, yol)
		return diffSonucu{satirlar: satirlar, err: err}
	}
}

func dosyaListesiCmd(calismaDir string) tea.Cmd {
	return func() tea.Msg {
		liste, err := dosyaListesiOku(calismaDir)
		return dosyaListesiSonucu{liste: liste, err: err}
	}
}

func dosyaAcCmd(calismaDir, rel string) tea.Cmd {
	return func() tea.Msg {
		icerik, err := dosyaIcerigiOku(calismaDir, rel)
		return dosyaAcSonucu{rel: rel, icerik: icerik, err: err}
	}
}

func dosyaKaydetCmd(calismaDir, rel, icerik string) tea.Cmd {
	return func() tea.Msg {
		return dosyaKaydetSonucu{err: dosyaIcerigiYaz(calismaDir, rel, icerik)}
	}
}

func dosyaSilCmd(calismaDir, rel string) tea.Cmd {
	return func() tea.Msg {
		return dosyaSilSonucu{rel: rel, err: dosyaSil(calismaDir, rel)}
	}
}

func disaAktarCmd(dataDir, projeID, projeAd, hedefKlasor string) tea.Cmd {
	return func() tea.Msg {
		yol, err := projeDisaAktar(dataDir, projeID, projeAd, hedefKlasor)
		return disaAktarSonucu{yol: yol, err: err}
	}
}

func pushCmd(c istemci, dataDir string, g Grup, p Proje, hedefUye Uye, bilinenBaslangic string) tea.Cmd {
	return func() tea.Msg {
		n, err := pushYap(c, dataDir, g, p, hedefUye, bilinenBaslangic)
		return pushSonucu{hedefAd: hedefUye.Ad, nesneSayisi: n, err: err}
	}
}

func pullIsteCmd(c istemci, hedefUye Uye, g Grup, p Proje, kendi Uye) tea.Cmd {
	return func() tea.Msg {
		return basitSonuc{islem: "pull_iste", err: pullIste(c, hedefUye, g, p, kendi)}
	}
}

func birlestirCmd(dataDir, projeID, calismaDir, yerelHead, uzakHead, yazar string) tea.Cmd {
	return func() tea.Msg {
		yerelCommit, err := commitOku(dataDir, projeID, yerelHead)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		durum, err := durumHesapla(dataDir, projeID, calismaDir, yerelCommit.Agac)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		if !durum.temizMi() {
			return birlestirSonucu{projeID: projeID, err: fmt.Errorf("önce mevcut değişiklikleri kaydet")}
		}

		uzakCommit, err := commitOku(dataDir, projeID, uzakHead)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: fmt.Errorf("uzak commit bulunamadı — önce Gönder/Çek ile nesneleri al: %w", err)}
		}

		temel := ortakAta(dataDir, projeID, yerelHead, uzakHead)
		if temel == uzakHead {
			return birlestirSonucu{projeID: projeID, err: fmt.Errorf("zaten güncel")}
		}
		if temel == yerelHead {
			if err := dalaGec(dataDir, projeID, calismaDir, uzakCommit.Agac); err != nil {
				return birlestirSonucu{projeID: projeID, err: err}
			}
			return birlestirSonucu{projeID: projeID, yeniCommit: uzakHead}
		}

		temelTree, err := headTreeOf(dataDir, projeID, temel)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		catismalar, err := birlestirmeUygula(dataDir, projeID, calismaDir, temelTree, yerelCommit.Agac, uzakCommit.Agac)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		if len(catismalar) > 0 {
			_ = birlesmeIsaretle(dataDir, projeID, uzakHead)
			return birlestirSonucu{projeID: projeID, catismalar: catismalar, uzakHash: uzakHead}
		}
		yeniTree, _, _, err := agacOlustur(dataDir, projeID, calismaDir)
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		yeniHash, err := commitOlustur(dataDir, projeID, yeniTree, []string{yerelHead, uzakHead}, yazar, "Birleştirme: "+kisaHash(uzakHead))
		if err != nil {
			return birlestirSonucu{projeID: projeID, err: err}
		}
		return birlestirSonucu{projeID: projeID, yeniCommit: yeniHash}
	}
}

// ─── DM protokol zarfı işleme ───────────────────────────────────────────────

// zarfIsle, gelen bir kontrol mesajını işler. verifiedSender, bu DM'in
// GERÇEKTEN geldiği, libp2p bağlantısıyla kriptografik olarak doğrulanmış
// peer ID'sidir (bkz. pluginapi.DMMsg.VerifiedPeerID) — z.Payload içindeki
// herhangi bir "gönderen" alanı ise yalnızca göndericinin kendi beyanıdır ve
// tek başına ASLA yetki kanıtı olarak kullanılmamalıdır. Her dal, yetki
// kontrolünün yanı sıra, payload'ın iddia ettiği kimliğin verifiedSender'la
// eşleştiğini de doğrular; aksi halde herhangi bir peer, payload'a başka
// birinin (ör. gerçek kurucunun) peer ID'sini yazarak onun yerine geçebilir.
//
// alindi, mesajın bu düğüme ulaştığı andır (AND'ın kendi saatiyle; gönderen
// bunu belirleyemez). DeltaX kapalıyken gelen mesajlar geçmişten sonradan
// işlendiği için bazı türler yaşına göre atlanır.
func (m model) zarfIsle(z kontrolZarfi, verifiedSender string, alindi time.Time) (model, tea.Cmd) {
	switch z.Tip {
	case mDavet, mUyeGuncelle:
		var dp davetPayload
		if json.Unmarshal(z.Payload, &dp) != nil {
			return m, nil
		}
		if dp.Gonderen.PeerID != verifiedSender {
			return m, nil
		}
		if g, err := grupOku(m.dataDir, z.GrupID); err == nil {
			// Zaten bildiğimiz bir grup için gelen güncelleme, yalnızca zaten
			// güvendiğimiz bir yetkilinin (kurucu ya da yönetici) peer
			// ID'siyle eşleşiyorsa kabul edilir.
			if !g.yetkiliMi(dp.Gonderen.PeerID) {
				return m, nil
			}
			g.Uyeler = dp.Uyeler
			g.Yoneticiler = dp.Yoneticiler
			g.Ad = dp.GrupAd
			_ = grupKaydet(m.dataDir, g)
			if m.aktifGrup != nil && m.aktifGrup.ID == g.ID {
				*m.aktifGrup = g
			}
			m = m.bildirGuncelle("\""+safe(dp.GrupAd)+"\" grubunun üye listesi güncellendi.", false)
			return m, nil
		}
		if z.Tip == mUyeGuncelle {
			return m, nil
		}
		// Davet eden, bu kadar eski bir davetin kabulünü artık tanımaz.
		if time.Since(alindi) > davetGecerlilik {
			return m, nil
		}
		for _, d := range m.bekleyenDavetler {
			if d.GrupID == z.GrupID {
				return m, nil
			}
		}
		m.bekleyenDavetler = append(m.bekleyenDavetler, bekleyenDavet{GrupID: z.GrupID, GonderenPeer: dp.Gonderen.PeerID, Payload: dp})
		_ = davetleriKaydet(m.dataDir, m.bekleyenDavetler)
		m = m.bildirGuncelle("Yeni grup daveti: "+safe(dp.GrupAd)+" (davet eden: "+safe(dp.Gonderen.Ad)+")", false)
		return m, nil

	case mDavetYanit:
		var yp davetYanitPayload
		if json.Unmarshal(z.Payload, &yp) != nil {
			return m, nil
		}
		if yp.Uye.PeerID != verifiedSender {
			return m, nil
		}
		// Bu grup için gerçekten bu peer'a bir davet gönderdiğimizi
		// doğrula — aksi halde davet edilmemiş biri bile, kendi gerçek
		// kimliğiyle "kabul ediyorum" diyerek gruba katılabilirdi (hiçbir
		// zaman gönderilmemiş bir daveti "kabul etmek"). bkz. davetDefteri.
		if !m.gonderilenDavetler.gecerli(z.GrupID, verifiedSender, alindi) {
			return m, nil
		}
		g, err := grupOku(m.dataDir, z.GrupID)
		if err != nil || !g.yetkiliMi(m.identity.PeerID) {
			return m, nil
		}
		// Yanıt ne olursa olsun davet kullanılmıştır: reddeden biri sonradan
		// aynı davetle "kabul" edemez.
		m.gonderilenDavetler.sil(z.GrupID, verifiedSender)
		m.gonderilenDavetler.kaydet(m.dataDir)
		if !yp.Kabul {
			m = m.bildirGuncelle(safe(yp.Uye.Ad)+" \""+safe(g.Ad)+"\" grubuna davetini reddetti.", false)
			return m, nil
		}
		g.uyeEkleGuncelle(yp.Uye)
		_ = grupKaydet(m.dataDir, g)
		if m.aktifGrup != nil && m.aktifGrup.ID == g.ID {
			*m.aktifGrup = g
		}
		m = m.bildirGuncelle(safe(yp.Uye.Ad)+" \""+safe(g.Ad)+"\" grubuna katıldı.", false)
		return m, rosterYayinCmd(m.client, g, m.kendiUye())

	case mPushBildir:
		var bp pushBildirPayload
		if json.Unmarshal(z.Payload, &bp) == nil {
			m = m.bildirGuncelle(fmt.Sprintf("Bir üye değişiklik gönderiyor (%s, %d nesne) — kabul edilince Uzak Dallar'da görünecek.", safe(bp.Dal), bp.NesneSayisi), false)
		}
		return m, nil

	case mPullIstek:
		var pp pullIstekPayload
		if json.Unmarshal(z.Payload, &pp) != nil {
			return m, nil
		}
		if pp.Isteyen.PeerID != verifiedSender {
			return m, nil
		}
		// Gönderim, isteyenin şu an çevrimiçi olmasını gerektirir: DeltaX
		// kapalıyken gelmiş eski bir istek yanıtlanmaz.
		if time.Since(alindi) > pullGecerlilik {
			return m, nil
		}
		g, gerr := grupOku(m.dataDir, z.GrupID)
		p, perr := projeOku(m.dataDir, z.ProjeID)
		if gerr != nil || perr != nil || !g.uyeMi(pp.Isteyen.PeerID) {
			return m, nil
		}
		return m, pushCmd(m.client, m.dataDir, g, p, pp.Isteyen, pp.BilinenHash)

	case mAyril:
		var ap ayrilPayload
		if json.Unmarshal(z.Payload, &ap) != nil {
			return m, nil
		}
		// ayrilBildir yalnızca kendi ayrılışını bildirmek için kullanılır
		// (bkz. sync.go'nun doc yorumu) — bu yüzden ap.Uye, DM'i GERÇEKTEN
		// gönderen peer'la eşleşmelidir; aksi halde herhangi bir peer,
		// başka bir üyeyi keyfi olarak "ayrılmış" gösterip attırabilirdi.
		if ap.Uye.PeerID != verifiedSender {
			return m, nil
		}
		g, err := grupOku(m.dataDir, z.GrupID)
		if err != nil || !g.yetkiliMi(m.identity.PeerID) || !g.uyeMi(ap.Uye.PeerID) {
			return m, nil
		}
		g.uyeCikar(ap.Uye.PeerID)
		g.yoneticiDurumunuDegistir(ap.Uye, false) // ayrılan bir yöneticiyse listeden de düşür
		_ = grupKaydet(m.dataDir, g)
		if m.aktifGrup != nil && m.aktifGrup.ID == g.ID {
			*m.aktifGrup = g
		}
		m = m.bildirGuncelle(safe(ap.Uye.Ad)+" \""+safe(g.Ad)+"\" grubundan ayrıldı.", false)
		return m, rosterYayinCmd(m.client, g, m.kendiUye())

	case mGrupKapat:
		var kp grupKapatPayload
		if json.Unmarshal(z.Payload, &kp) != nil {
			return m, nil
		}
		if kp.Gonderen.PeerID != verifiedSender {
			return m, nil
		}
		g, err := grupOku(m.dataDir, z.GrupID)
		if err != nil || !g.yetkiliMi(kp.Gonderen.PeerID) {
			return m, nil
		}
		_ = grupSil(m.dataDir, g.ID)
		if m.aktifGrup != nil && m.aktifGrup.ID == g.ID {
			m.aktifGrup = nil
			m.ekran = ekGrupListesi
		}
		m.gruplar = gruplariListele(m.dataDir)
		m.notice = "\"" + safe(g.Ad) + "\" grubu kurucusu tarafından kapatıldı."
		m.isErr = true
		return m, nil
	}
	return m, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

// diffMetni, farkı değişmeyen uzun bölümleri katlanmış olarak yazar (bkz.
// diffKatla): eklenen satırlar yeşil "+", silinenler kırmızı "−", bağlam
// soluk, katlanan bölümler "⋯ N satır aynı".
func diffMetni(satirlar []DiffSatiri) string {
	var sb strings.Builder
	for _, s := range diffKatla(satirlar, diffBaglam) {
		switch s.Tip {
		case '+':
			sb.WriteString(stAdd.Render("+ " + safe(s.Metin)))
		case '-':
			sb.WriteString(stDel.Render("− " + safe(s.Metin)))
		case '@':
			sb.WriteString(ui.S.Subtle.Render("  " + s.Metin))
		default:
			sb.WriteString(stDim.Render("  " + safe(s.Metin)))
		}
		sb.WriteByte('\n')
	}
	if len(satirlar) == 0 {
		sb.WriteString(stMuted.Render("İki taraf da boş: gösterilecek fark yok.") + "\n")
	}
	return sb.String()
}
