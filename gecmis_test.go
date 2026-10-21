package main

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/internal/pluginapi"
)

const (
	pKurucu   = "12D3KooWKurucuAbCdEfGhJkLmNpQrStUvWxYz123456789abcd"
	pDavetli  = "12D3KooWDavetAbCdEfGhJkLmNpQrStUvWxYz123456789abcde"
	pYonetici = "12D3KooWYonetAbCdEfGhJkLmNpQrStUvWxYz123456789abcde"
)

// sahteIstemci, AND'ın eklenti API'sinin bellekteki taklididir.
type sahteIstemci struct {
	mu         sync.Mutex
	gecmis     []pluginapi.DMRecord
	gecmisErr  error
	pollSayisi int
	// istenenler: DMHistoryFrom'a verilen kimlikler.
	istenenler []string
	// dogrudan: SendDM ile, sirali: SendDMQueued ile gönderilenler ("peer|metin").
	dogrudan, sirali []string
}

// DMHistoryFrom, AND gibi geçmişi id'li kayıttan (dahil) itibaren keser;
// kayıt yoksa tamamını döner.
func (s *sahteIstemci) DMHistoryFrom(id string) ([]pluginapi.DMRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.istenenler = append(s.istenenler, id)
	k := s.gecmis
	for i := len(k) - 1; i >= 0 && id != ""; i-- {
		if k[i].ID == id {
			k = k[i:]
			break
		}
	}
	return append([]pluginapi.DMRecord(nil), k...), s.gecmisErr
}

func (s *sahteIstemci) PollDM() ([]pluginapi.DMMsg, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollSayisi++
	return nil, nil
}

func (s *sahteIstemci) SendDM(peerID, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dogrudan = append(s.dogrudan, peerID+"|"+message)
	return nil
}

func (s *sahteIstemci) SendDMQueued(peerID, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sirali = append(s.sirali, peerID+"|"+message)
	return nil
}

func (s *sahteIstemci) SendFile(string, string) error                    { return nil }
func (s *sahteIstemci) PollFile() ([]pluginapi.FileMsg, error)           { return nil, nil }
func (s *sahteIstemci) PollConsent() ([]pluginapi.FileConsentReq, error) { return nil, nil }
func (s *sahteIstemci) RespondConsent(string, bool) error                { return nil }
func (s *sahteIstemci) Peers() ([]pluginapi.PeerInfo, error)             { return nil, nil }

// gelsin, AND'ın geçmişine gelen bir mesaj ekler.
func (s *sahteIstemci) gelsin(id, peerID, metin string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gecmis = append(s.gecmis, pluginapi.DMRecord{ID: id, PeerID: peerID, Text: metin, At: at})
}

func zarfMetni(t *testing.T, tip mesajTipi, grupID string, payload any) string {
	t.Helper()
	p, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	metin, err := zarfKodla(kontrolZarfi{Tip: tip, GrupID: grupID, Payload: p})
	if err != nil {
		t.Fatal(err)
	}
	return metin
}

// gecmisIsle, geçmişi bir kez okuyup işler (Init'in ve her turun yaptığı gibi).
func gecmisIsle(t *testing.T, m model) (model, tea.Cmd) {
	t.Helper()
	mm, cmd := m.Update(gecmisCek(m.client, m.imlec.ID))
	return mm.(model), cmd
}

// calistir, cmd'yi ve (Batch ise) alt komutlarını çalıştırır.
func calistir(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if b, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range b {
			calistir(c)
		}
	}
}

func testGrubu(t *testing.T, dataDir string, uyeler ...Uye) Grup {
	t.Helper()
	g := Grup{ID: "grup1", Ad: "Proje Ekibi", Sir: mustSir(t), KurucuAd: "kurucu", KurucuPeer: pKurucu,
		Uyeler: append([]Uye{{Ad: "kurucu", PeerID: pKurucu}}, uyeler...)}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGecmis_IlkCalistirmadaEskiMesajlarIslenmez(t *testing.T) {
	dataDir := t.TempDir()
	c := &sahteIstemci{}
	kurucu := Uye{Ad: "kurucu", PeerID: pKurucu}
	c.gelsin("k1", pKurucu, zarfMetni(t, mDavet, "grup1", davetPayload{GrupAd: "Eski", Sir: mustSir(t), Kurucu: kurucu, Uyeler: []Uye{kurucu}, Gonderen: kurucu}),
		time.Now().UTC().Add(-time.Minute))

	m, _ := gecmisIsle(t, newModel(c, pluginapi.IdentityInfo{PeerID: pDavetli}, dataDir))
	if len(m.bekleyenDavetler) != 0 {
		t.Fatal("eski sürümün canlı olarak işlediği mesajlar ilk çalıştırmada yeniden işlenmemeli")
	}
	if im := imleciOku(dataDir); im.ID != "k1" {
		t.Fatalf("imleç geçmişin sonuna konmalı: %+v", im)
	}
}

func TestGecmis_KapaliykenGelenDavetSonradanIslenir(t *testing.T) {
	dataDir := t.TempDir()
	c := &sahteIstemci{}
	id := pluginapi.IdentityInfo{PeerID: pDavetli}
	gecmisIsle(t, newModel(c, id, dataDir)) // ilk çalıştırma: geçmiş boş

	// DeltaX kapalıyken bir davet gelir.
	kurucu := Uye{Ad: "kurucu", PeerID: pKurucu}
	c.gelsin("k1", pKurucu, zarfMetni(t, mDavet, "grup1", davetPayload{GrupAd: "Proje Ekibi", Sir: mustSir(t), Kurucu: kurucu, Uyeler: []Uye{kurucu}, Gonderen: kurucu}),
		time.Now().UTC().Add(time.Second))

	m, _ := gecmisIsle(t, newModel(c, id, dataDir)) // yeniden açılış
	if len(m.bekleyenDavetler) != 1 || m.bekleyenDavetler[0].Payload.GrupAd != "Proje Ekibi" {
		t.Fatalf("kapalıyken gelen davet açılınca görünmeli: %+v", m.bekleyenDavetler)
	}
	if im := imleciOku(dataDir); im.ID != "k1" {
		t.Fatalf("imleç işlenen son kayda ilerlemeli: %+v", im)
	}
}

func TestGecmis_HerTurdaYalnizcaYenileriIster(t *testing.T) {
	dataDir := t.TempDir()
	c := &sahteIstemci{}
	c.gelsin("k1", pKurucu, "selam", time.Now().UTC())
	m, _ := gecmisIsle(t, newModel(c, pluginapi.IdentityInfo{PeerID: pDavetli}, dataDir))

	kurucu := Uye{Ad: "kurucu", PeerID: pKurucu}
	c.gelsin("k2", pKurucu, zarfMetni(t, mDavet, "grup1", davetPayload{GrupAd: "Proje Ekibi", Sir: mustSir(t), Kurucu: kurucu, Uyeler: []Uye{kurucu}, Gonderen: kurucu}),
		time.Now().UTC().Add(time.Second))
	m, _ = gecmisIsle(t, m)
	if got := c.istenenler[len(c.istenenler)-1]; got != "k1" {
		t.Fatalf("geçmiş imlecin kaydından itibaren istenmeli: %q", got)
	}
	if len(m.bekleyenDavetler) != 1 {
		t.Fatalf("kesitteki yeni kayıt işlenmeli: %+v", m.bekleyenDavetler)
	}
	m, _ = gecmisIsle(t, m)
	if len(m.bekleyenDavetler) != 1 || c.istenenler[len(c.istenenler)-1] != "k2" {
		t.Fatalf("aynı kayıt ikinci kez işlenmemeli: %+v", m.bekleyenDavetler)
	}
}

func TestGecmis_DavetKabuluYenidenAcilincaTaninir(t *testing.T) {
	dataDir := t.TempDir()
	g := testGrubu(t, dataDir)
	c := &sahteIstemci{}
	id := pluginapi.IdentityInfo{Name: "kurucu", PeerID: pKurucu}
	m, _ := gecmisIsle(t, newModel(c, id, dataDir))

	// Davet ekranından davet gönderilir.
	m.aktifGrup, m.ekran = &g, ekUyeDavet
	m.uyeDavetIn.SetValue(pDavetli)
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	calistir(cmd)
	if mm.(model).isErr {
		t.Fatalf("davet gönderilemedi: %s", mm.(model).notice)
	}
	if len(c.sirali) != 1 || !strings.HasPrefix(c.sirali[0], pDavetli+"|"+protokolOnEki) {
		t.Fatalf("davet giden kutusuyla gitmeli: sirali=%v dogrudan=%v", c.sirali, c.dogrudan)
	}

	// DeltaX kapalıyken kabul gelir; yeniden açılınca tanınmalı.
	davetli := Uye{Ad: "davetli", PeerID: pDavetli}
	c.gelsin("y1", pDavetli, zarfMetni(t, mDavetYanit, g.ID, davetYanitPayload{Kabul: true, Uye: davetli}), time.Now().UTC().Add(time.Second))
	m2, _ := gecmisIsle(t, newModel(c, id, dataDir))
	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sonra.uyeMi(pDavetli) {
		t.Fatalf("kapalıyken gelen kabul uygulanmalı: %s", m2.notice)
	}
	if davetDefteriniOku(dataDir, time.Now()).gecerli(g.ID, pDavetli, time.Now()) {
		t.Fatal("kullanılan davet defterden düşmeli")
	}
}

func TestGecmis_EskiANDdaCanliDinlemeyeDuser(t *testing.T) {
	c := &sahteIstemci{gecmisErr: &pluginapi.APIError{Status: 404}}
	_, cmd := gecmisIsle(t, newModel(c, pluginapi.IdentityInfo{PeerID: pDavetli}, t.TempDir()))
	if cmd == nil {
		t.Fatal("bir sonraki komut dönmeli")
	}
	if _, ok := cmd().(dmPollResultMsg); !ok || c.pollSayisi != 1 {
		t.Fatalf("geçmişi olmayan AND'da canlı dinlemeye düşmeli (poll=%d)", c.pollSayisi)
	}
}

func TestImlec_SirayaGoreIlerler(t *testing.T) {
	t0 := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	kayitlar := []pluginapi.DMRecord{
		{ID: "a", At: t0},
		{ID: "b", At: t0.Add(time.Second)},
		{ID: "c", At: t0.Add(-time.Hour)}, // saat geri alınmış
	}
	idler := func(k []pluginapi.DMRecord) string {
		var s []string
		for _, r := range k {
			s = append(s, r.ID)
		}
		return strings.Join(s, ",")
	}
	for _, tc := range []struct {
		ad       string
		im       dmImleci
		beklenen string
	}{
		{"kayıt yerinde: saat geri alınsa da sıraya göre", dmImleci{ID: "a", Zaman: t0}, "b,c"},
		{"son kayıt", dmImleci{ID: "c", Zaman: t0.Add(-time.Hour)}, ""},
		{"kayıt geçmişten düşmüş: zamana göre", dmImleci{ID: "silinmis", Zaman: t0}, "b,c"},
		{"kimlik göndermeyen AND", dmImleci{Zaman: t0.Add(time.Second)}, ""},
	} {
		if got := idler(tc.im.yeniler(kayitlar)); got != tc.beklenen {
			t.Errorf("%s: %q; %q beklenirdi", tc.ad, got, tc.beklenen)
		}
	}
}

func TestZarfIsle_EskiCekIstegiYanitlanmaz(t *testing.T) {
	dataDir := t.TempDir()
	isteyen := Uye{Ad: "isteyen", PeerID: pDavetli}
	g := testGrubu(t, dataDir, isteyen)
	p, err := projeOlustur(dataDir, g.ID, "proje")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(pullIstekPayload{Dal: "ana", Isteyen: isteyen})
	z := kontrolZarfi{Tip: mPullIstek, GrupID: g.ID, ProjeID: p.ID, Payload: payload}
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: pKurucu}}

	if _, cmd := m.zarfIsle(z, isteyen.PeerID, time.Now().Add(-2*time.Hour)); cmd != nil {
		t.Fatal("DeltaX kapalıyken gelmiş eski bir çek isteği yanıtlanmamalı")
	}
	if _, cmd := m.zarfIsle(z, isteyen.PeerID, time.Now()); cmd == nil {
		t.Fatal("yeni bir çek isteği yanıtlanmalı")
	}
}

func TestZarfIsle_ReddedilenDavetKullanilmisSayilir(t *testing.T) {
	dataDir := t.TempDir()
	g := testGrubu(t, dataDir)
	davetli := Uye{Ad: "davetli", PeerID: pDavetli}
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: pKurucu},
		gonderilenDavetler: davetDefteri{g.ID: {pDavetli: time.Now()}}}

	red, _ := json.Marshal(davetYanitPayload{Kabul: false, Uye: davetli})
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mDavetYanit, GrupID: g.ID, Payload: red}, pDavetli, time.Now())
	if !strings.Contains(m.notice, "reddetti") {
		t.Fatalf("ret bildirilmeli: %q", m.notice)
	}
	kabul, _ := json.Marshal(davetYanitPayload{Kabul: true, Uye: davetli})
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mDavetYanit, GrupID: g.ID, Payload: kabul}, pDavetli, time.Now())
	if sonra, _ := grupOku(dataDir, g.ID); sonra.uyeMi(pDavetli) {
		t.Fatal("reddedilen davet sonradan kabul edilememeli")
	}
}

func TestDavetDefteri_SuresiDolanDuser(t *testing.T) {
	dataDir := t.TempDir()
	simdi := time.Now()
	var d davetDefteri
	d.ekle("g1", pDavetli, simdi.Add(-davetGecerlilik-time.Hour))
	d.ekle("g1", pYonetici, simdi)
	d.kaydet(dataDir)

	okunan := davetDefteriniOku(dataDir, simdi)
	if okunan.gecerli("g1", pDavetli, simdi) || !okunan.gecerli("g1", pYonetici, simdi) {
		t.Fatalf("süresi dolan düşmeli, geçerli olan kalmalı: %+v", okunan)
	}
	// Geçerlilik yanıtın alındığı ana göre ölçülür: süre içinde gelmiş bir
	// yanıt, DeltaX sonradan açılsa da tanınır.
	if !okunan.gecerli("g1", pYonetici, simdi.Add(davetGecerlilik-time.Minute)) ||
		okunan.gecerli("g1", pYonetici, simdi.Add(davetGecerlilik+time.Minute)) {
		t.Fatal("geçerlilik yanıtın alındığı ana göre ölçülmeli")
	}
}

func TestDavetEt_GecersizGirdilerReddedilir(t *testing.T) {
	dataDir := t.TempDir()
	g := testGrubu(t, dataDir, Uye{Ad: "üye", PeerID: pYonetici})
	m := newModel(&sahteIstemci{}, pluginapi.IdentityInfo{Name: "kurucu", PeerID: pKurucu}, dataDir)
	m.aktifGrup, m.ekran = &g, ekUyeDavet
	for _, tc := range []struct{ girdi, beklenen string }{
		{"merhaba", "Peer ID değil"},
		{pKurucu, "Kendini"},
		{pYonetici, "zaten"},
	} {
		m.uyeDavetIn.SetValue(tc.girdi)
		mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		got := mm.(model)
		if cmd != nil || !got.isErr || !strings.Contains(got.notice, tc.beklenen) {
			t.Errorf("%q: %q (hata=%v); %q içermeli", tc.girdi, got.notice, got.isErr, tc.beklenen)
		}
	}
	if len(davetDefteriniOku(dataDir, time.Now())) != 0 {
		t.Fatal("reddedilen girdiler davet defterine yazılmamalı")
	}
}

func TestBildirimler_GidenKutusuylaGider(t *testing.T) {
	c := &sahteIstemci{}
	ben := Uye{Ad: "kurucu", PeerID: pKurucu}
	g := Grup{ID: "grup1", Ad: "Proje Ekibi", Sir: "sir", KurucuAd: ben.Ad, KurucuPeer: pKurucu,
		Yoneticiler: []Uye{{Ad: "yönetici", PeerID: pYonetici}},
		Uyeler:      []Uye{ben, {Ad: "davetli", PeerID: pDavetli}, {Ad: "yönetici", PeerID: pYonetici}, {Ad: "tekrar", PeerID: pDavetli}}}

	hatalar := uyeGuncellemesiYayinla(c, g, ben)                           // davetli + yönetici (tekrar yok)
	hatalar = append(hatalar, grupKapatBildir(c, g, ben)...)               // davetli + yönetici
	hatalar = append(hatalar, ayrilBildir(c, g, Uye{PeerID: pDavetli})...) // kurucu + yönetici
	if err := davetGonder(c, pDavetli, g, ben); err != nil {
		hatalar = append(hatalar, err)
	}
	if err := davetYanitGonder(c, pKurucu, g.ID, true, Uye{PeerID: pDavetli}); err != nil {
		hatalar = append(hatalar, err)
	}
	if len(hatalar) != 0 {
		t.Fatal(hatalar)
	}
	if len(c.sirali) != 8 || len(c.dogrudan) != 0 {
		t.Fatalf("grup durumunu değiştiren mesajlar giden kutusuyla gitmeli: %d sıralı, %d doğrudan", len(c.sirali), len(c.dogrudan))
	}

	// Çek isteği iki tarafın da çevrimiçi olmasını gerektirir: doğrudan gider.
	if err := pullIste(c, Uye{PeerID: pDavetli}, g, Proje{ID: "p1", AktifDal: "ana", Dallar: map[string]string{"ana": ""}}, ben); err != nil {
		t.Fatal(err)
	}
	if len(c.dogrudan) != 1 {
		t.Fatalf("çek isteği doğrudan gitmeli: %v", c.dogrudan)
	}
}
