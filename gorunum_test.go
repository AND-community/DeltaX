package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lucian95511/and/internal/pluginapi"
)

const pUye = "12D3KooWUyeAbCdEfGhJkLmNpQrStUvWxYz123456789abcdefg"

func tusBas(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// bas, tuşa basar ve dönen komutları bir kez çalıştırıp sonuçlarını modele
// iletir (spinner tıkları ve yeniden yoklamalar izlenmez).
func bas(t *testing.T, m model, tuslar ...string) model {
	t.Helper()
	for _, k := range tuslar {
		mm, cmd := m.Update(tusBas(k))
		m = isle(t, mm.(model), cmd)
	}
	return m
}

func isle(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	for _, msg := range calistirTopla(cmd) {
		switch msg.(type) {
		case spinner.TickMsg, peersResultMsg:
			continue // tık zinciri ve eş yoklaması testte izlenmez
		}
		mm, _ := m.Update(msg)
		m = mm.(model)
	}
	return m
}

// calistirTopla, cmd'yi (Batch ise alt komutlarıyla) çalıştırıp mesajları toplar.
func calistirTopla(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, calistirTopla(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// ekranMetni, görünümü renksiz metin olarak döner ve her satırın terminal
// genişliğine sığdığını denetler.
func ekranMetni(t *testing.T, m model) string {
	t.Helper()
	v := m.View()
	for i, satir := range strings.Split(v, "\n") {
		if w := lipgloss.Width(satir); w > m.w {
			t.Errorf("%v ekranında %d. satır %d sütun, terminal %d: %q", m.ekran, i+1, w, m.w, ansi.Strip(satir))
		}
	}
	return ansi.Strip(v)
}

// ornekProje, iki kayıtlı bir proje ve bir kaydedilmemiş değişiklikle hazır bir model kurar.
func ornekProje(t *testing.T, w, h int) (model, string, string) {
	t.Helper()
	simdi = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { simdi = time.Now })
	dataDir := t.TempDir()
	g := testGrubu(t, dataDir, Uye{Ad: "ayşe", PeerID: pUye})
	p, err := projeOlustur(dataDir, g.ID, "Web Sitesi")
	if err != nil {
		t.Fatal(err)
	}
	calisma := projeCalismaDizini(dataDir, p.ID)
	yazDosya(t, calisma, "index.html", "<h1>Merhaba</h1>\n")
	yazDosya(t, calisma, "stil.css", "body{}\n")
	k1 := kaydet(t, dataDir, p.ID, calisma, "", "İlk sürüm")
	yazDosya(t, calisma, "index.html", "<h1>Merhaba dünya</h1>\n")
	k2 := kaydet(t, dataDir, p.ID, calisma, k1, "Başlık güncellendi")
	p.Dallar["ana"] = k2
	p.Dallar["deneme"] = k1
	p.Uzaklar = map[string]string{pUye + "/ana": k2}
	if err := projeKaydet(dataDir, p); err != nil {
		t.Fatal(err)
	}
	yazDosya(t, calisma, "stil.css", "body{color:red}\n")
	yazDosya(t, calisma, "yeni.js", "console.log(1)\n")

	m := newModel(&sahteIstemci{}, pluginapi.IdentityInfo{Name: "kurucu", PeerID: pKurucu}, dataDir)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = mm.(model)
	m.cevrimici = map[string]bool{pUye: true}
	// Ana menü → Gruplarım → grup → proje.
	m = bas(t, m, "enter", "enter", "enter")
	if m.ekran != ekProjePaneli {
		t.Fatalf("proje paneline gelinemedi: %v", m.ekran)
	}
	return m, k1, k2
}

func TestGorunum_EkranlarCizilir(t *testing.T) {
	for _, boyut := range [][2]int{{80, 24}, {140, 40}} {
		m, _, _ := ornekProje(t, boyut[0], boyut[1])
		ekranlar := map[string]model{}
		ekranlar["proje"] = m
		ekranlar["degisiklikler"] = bas(t, m, "d")
		ekranlar["gunluk"] = bas(t, m, "l")
		ekranlar["kayit"] = bas(t, m, "l", "enter")
		ekranlar["kayit-fark"] = bas(t, m, "l", "enter", "enter")
		ekranlar["dallar"] = bas(t, m, "b")
		ekranlar["uzak"] = bas(t, m, "m")
		ekranlar["gonder"] = bas(t, m, "p")
		grup := bas(t, m, "esc")
		ekranlar["grup"] = grup
		ekranlar["davet"] = bas(t, grup, "i")
		ana := bas(t, grup, "esc", "esc")
		ekranlar["ana"] = ana

		dokum := os.Getenv("DELTAX_EKRAN_DOKUM")
		for ad, e := range ekranlar {
			metin := ekranMetni(t, e)
			if !strings.Contains(metin, "DeltaX") {
				t.Errorf("%s: başlık çubuğu yok", ad)
			}
			if dokum != "" {
				_ = os.WriteFile(filepath.Join(dokum, ad+"_"+strconv.Itoa(boyut[0])+".txt"), []byte(metin), 0o600)
			}
		}

		p := ekranMetni(t, ekranlar["proje"])
		for _, beklenen := range []string{"⎇ ana", "Başlık güncellendi", "~1 değişen", "+1 yeni", "Değişiklikler", "Günlük"} {
			if !strings.Contains(p, beklenen) {
				t.Errorf("proje paneli %q içermeli:\n%s", beklenen, p)
			}
		}
		if d := ekranMetni(t, ekranlar["degisiklikler"]); !strings.Contains(d, "stil.css") || !strings.Contains(d, "yeni.js") {
			t.Errorf("değişiklikler listesi:\n%s", d)
		}
		if k := ekranMetni(t, ekranlar["kayit"]); !strings.Contains(k, "Başlık güncellendi") || !strings.Contains(k, "index.html") {
			t.Errorf("kayıt ayrıntısı:\n%s", k)
		}
		if f := ekranMetni(t, ekranlar["kayit-fark"]); !strings.Contains(f, "+1 −1") || !strings.Contains(f, "Merhaba dünya") {
			t.Errorf("kayıt farkı:\n%s", f)
		}
		if u := ekranMetni(t, ekranlar["uzak"]); !strings.Contains(u, "ayşe") || !strings.Contains(u, "güncel") {
			t.Errorf("uzak dallar, üye adı ve güncellik:\n%s", u)
		}
	}
}

func TestAkis_DegisiklikGeriAlinir(t *testing.T) {
	m, _, _ := ornekProje(t, 100, 30)
	m = bas(t, m, "d")
	liste := degisiklikListesi(m.durum)
	if len(liste) != 2 || liste[0].Yol != "stil.css" || liste[1].Yol != "yeni.js" {
		t.Fatalf("önce değişen sonra yeni dosya: %+v", liste)
	}
	m = bas(t, m, "g")
	if m.ekran != ekOnay || !strings.Contains(ekranMetni(t, m), "stil.css") {
		t.Fatal("geri alma onay istemeli")
	}
	m = bas(t, m, "y")
	veri, _ := os.ReadFile(filepath.Join(m.calismaDir(), "stil.css"))
	if string(veri) != "body{}\n" || m.ekran != ekDurum {
		t.Fatalf("dosya son kayıttaki hâline dönmeli: %q (%v)", veri, m.ekran)
	}
	m = bas(t, m, "r", "down", "g", "y") // yeni.js: son kayıtta yok → silinir
	if _, err := os.Stat(filepath.Join(m.calismaDir(), "yeni.js")); !os.IsNotExist(err) {
		t.Fatal("yeni dosyanın geri alınması onu silmeli")
	}
}

func TestAkis_KayittanDalOlusturulur(t *testing.T) {
	m, k1, _ := ornekProje(t, 100, 30)
	m = bas(t, m, "l", "down", "enter") // ikinci sıradaki kayıt: İlk sürüm
	if m.ekran != ekKayit || m.kayit.hash != k1 {
		t.Fatalf("kayıt ayrıntısı açılmalı: %v %s", m.ekran, m.kayit.hash)
	}
	m = bas(t, m, "b", "e", "s", "k", "i", "enter")
	if m.ekran != ekDallar || m.aktifProje.Dallar["eski"] != k1 {
		t.Fatalf("dal seçilen kayıttan başlamalı: %v %+v", m.ekran, m.aktifProje.Dallar)
	}
	if p, _ := projeOku(m.dataDir, m.aktifProje.ID); p.Dallar["eski"] != k1 {
		t.Fatal("yeni dal diske yazılmalı")
	}
}

func TestAkis_DalSilinir(t *testing.T) {
	m, _, _ := ornekProje(t, 100, 30)
	m = bas(t, m, "b") // dallar: ana (etkin), deneme
	m = bas(t, m, "d")
	if m.ekran != ekDallar || !m.isErr {
		t.Fatal("etkin dal silinememeli")
	}
	m = bas(t, m, "down", "d", "y")
	if _, kaldi := m.aktifProje.Dallar["deneme"]; kaldi || len(m.dallar) != 1 {
		t.Fatalf("dal silinmeli: %+v", m.aktifProje.Dallar)
	}
}

func TestUzakDallar_YeniKayitSayisiGosterilir(t *testing.T) {
	m, k1, k2 := ornekProje(t, 100, 30)
	// Yerel dal eski kayıtta, üyenin ucu bir kayıt önde.
	m.aktifProje.Dallar["ana"] = k1
	m.aktifProje.Uzaklar[pUye+"/ana"] = k2
	_ = projeKaydet(m.dataDir, *m.aktifProje)
	m = bas(t, m, "esc", "enter") // panele yeniden gir: özet yeniden hesaplanır
	if p := ekranMetni(t, m); !strings.Contains(p, "1 yeni kayıt") {
		t.Fatalf("proje paneli birleştirilmemiş kaydı göstermeli:\n%s", p)
	}
	if u := ekranMetni(t, bas(t, m, "m")); !strings.Contains(u, "1 yeni kayıt") {
		t.Fatalf("uzak dallar:\n%s", u)
	}
}

func TestDavet_BagliANDDugumleriOnerilir(t *testing.T) {
	m, _, _ := ornekProje(t, 100, 30)
	m = bas(t, m, "esc")
	m.esler = []pluginapi.PeerInfo{
		{PeerID: pUye, AND: true},     // zaten üye
		{PeerID: pDavetli, AND: true}, // önerilir
		{PeerID: pYonetici},           // AND düğümü değil
	}
	m = bas(t, m, "i")
	if o := m.davetOnerileri(); len(o) != 1 || o[0].PeerID != pDavetli {
		t.Fatalf("yalnızca grupta olmayan AND düğümleri önerilmeli: %+v", o)
	}
	m = bas(t, m, "down")
	if m.uyeDavetIn.Value() != pDavetli {
		t.Fatalf("öneri seçilince Peer ID yazılmalı: %q", m.uyeDavetIn.Value())
	}
}
