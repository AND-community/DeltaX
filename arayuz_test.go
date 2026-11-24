package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/arayuz"
	"github.com/lucian95511/and/arayuz/arayuztest"
	"github.com/lucian95511/and/dil/diltest"
	"github.com/lucian95511/and/pluginapi"
)

// arayuzSurucusu, arayüz kipinde (merhaba almış) bir DeltaX modelini süren
// sürücüdür; veri dizini geçicidir.
func arayuzSurucusu(t *testing.T) *arayuztest.Surucu {
	t.Helper()
	m := newModel(&sahteIstemci{}, pluginapi.IdentityInfo{Name: "kurucu", PeerID: pKurucu}, t.TempDir())
	s := arayuztest.Yeni(m)
	s.Gonder(arayuz.Merhaba{Surum: 1, Dil: "tr"})
	return s
}

func modeli(s *arayuztest.Surucu) model { return s.Model().(model) }

// ekrandaVar, ekranın görünür metinlerinde aranan geçiyor mu?
func ekrandaVar(e arayuz.Ekran, aranan string) bool {
	return slices.ContainsFunc(arayuztest.Metinler(e), func(m string) bool { return strings.Contains(m, aranan) })
}

func listeOgeleri(t *testing.T, e arayuz.Ekran, id string) []arayuz.ListeOgesi {
	t.Helper()
	o, ok := arayuztest.Bul(e, id)
	if !ok {
		t.Fatalf("%q listesi ekranda yok", id)
	}
	return o.Ogeler
}

// grupKur, arayüzden bir grup kurar ve grubun kimliğini döner.
func grupKur(t *testing.T, s *arayuztest.Surucu, ad string) string {
	t.Helper()
	s.Tikla("yeniGrup")
	s.Yurut(s.FormGonder("grupFormu", map[string]string{"ad": ad}), 4)
	s.Tikla("yolAna")
	s.Yurut(nil, 1)
	for _, o := range listeOgeleri(t, s.Ekran(), "gruplar") {
		if o.Baslik == ad {
			return o.ID
		}
	}
	t.Fatalf("%q grubu listede yok", ad)
	return ""
}

func TestArayuz_GrupOlustur(t *testing.T) {
	s := arayuzSurucusu(t)
	if _, ok := arayuztest.Bul(s.Ekran(), "gruplar"); !ok {
		t.Fatal("girişte gruplar listesi yok")
	}
	s.Tikla("yeniGrup")
	e := s.Ekran()
	if e.Diyalog == nil {
		t.Fatal("yeni grup diyaloğu açılmadı")
	}
	if _, ok := arayuztest.Bul(e, "grupFormu"); !ok {
		t.Fatal("diyalogda grupFormu yok")
	}
	s.FormGonder("grupFormu", map[string]string{"ad": ""})
	if !ekrandaVar(s.Ekran(), "Grup adı boş olamaz.") {
		t.Errorf("boş ad uyarısı yok: %v", arayuztest.Metinler(s.Ekran()))
	}
	if id := grupKur(t, s, "Ekip"); id == "" {
		t.Fatal("grup kurulamadı")
	}
}

func TestArayuz_GrubaGir(t *testing.T) {
	s := arayuzSurucusu(t)
	id := grupKur(t, s, "Ekip")
	s.Yurut(s.Ac("gruplar", id), 2)
	e := s.Ekran()
	if len(e.Yol) != 2 || e.Yol[0].Metin != "DeltaX" || e.Yol[1].Metin != "Ekip" {
		t.Errorf("kırıntılar = %+v", e.Yol)
	}
	if _, ok := arayuztest.Bul(e, "projeler"); !ok {
		t.Error("grup görünümünde projeler listesi yok")
	}
	if uyeler := listeOgeleri(t, e, "uyeler"); len(uyeler) != 1 || uyeler[0].ID != pKurucu {
		t.Errorf("üyeler = %+v", uyeler)
	}
	s.Tikla("yolAna")
	if _, ok := arayuztest.Bul(s.Ekran(), "gruplar"); !ok {
		t.Error("ana kırıntı girişe dönmedi")
	}
}

func TestArayuz_ProjeOlustur(t *testing.T) {
	s := arayuzSurucusu(t)
	s.Ac("gruplar", grupKur(t, s, "Ekip"))
	s.Tikla("yeniProje")
	if d := s.Ekran().Diyalog; d == nil {
		t.Fatal("yeni proje diyaloğu açılmadı")
	}
	s.Yurut(s.FormGonder("projeFormu", map[string]string{"ad": "arayuz"}), 4)
	if o := listeOgeleri(t, s.Ekran(), "projeler"); len(o) != 1 || o[0].Baslik != "arayuz" {
		t.Errorf("projeler = %+v", o)
	}
	if s.Ekran().Diyalog != nil {
		t.Error("proje kurulunca diyalog kapanmadı")
	}
}

func TestArayuz_DavetGecersiz(t *testing.T) {
	s := arayuzSurucusu(t)
	s.Ac("gruplar", grupKur(t, s, "Ekip"))
	s.Tikla("uyeDavet")
	if s.Ekran().Diyalog == nil {
		t.Fatal("davet diyaloğu açılmadı")
	}
	s.FormGonder("davetFormu", map[string]string{"peer": "abc"})
	if !ekrandaVar(s.Ekran(), "Bu bir Peer ID değil") {
		t.Errorf("geçersiz Peer ID uyarısı yok: %v", arayuztest.Metinler(s.Ekran()))
	}
	s.DiyalogKapat()
	if s.Ekran().Diyalog != nil {
		t.Error("diyalog kapanmadı")
	}
}

func TestArayuz_GeriCikisYapmaz(t *testing.T) {
	s := arayuzSurucusu(t)
	for _, olay := range []string{arayuz.OlayGeri, arayuz.OlayDiyalogKapat} {
		cmd := s.Olay(arayuz.Olay{Olay: olay})
		if cmd != nil {
			if _, cik := cmd().(tea.QuitMsg); cik {
				t.Errorf("%s girişte eklentiyi kapattı", olay)
			}
		}
	}
	if _, ok := arayuztest.Bul(s.Ekran(), "gruplar"); !ok {
		t.Error("giriş kayboldu")
	}
}

func TestArayuz_GecerliEkran(t *testing.T) {
	s := arayuzSurucusu(t)
	arayuztest.Denetle(t, s.Ekran()) // boş giriş
	id := grupKur(t, s, "Ekip")
	arayuztest.Denetle(t, s.Ekran())
	s.Ac("gruplar", id)
	arayuztest.Denetle(t, s.Ekran())
	for _, d := range []string{"yeniProje", "uyeDavet", "yoneticiAta", "gruptanCik"} {
		s.Tikla(d)
		if s.Ekran().Diyalog == nil {
			t.Errorf("%s diyalog açmadı", d)
		}
		arayuztest.Denetle(t, s.Ekran())
		s.DiyalogKapat()
	}
	if m := modeli(s); m.ekran != ekGrupPaneli {
		t.Errorf("diyaloglar kapanınca ekran %v", m.ekran)
	}
}

// projeKur, arayüzden bir grup ve içinde bir proje kurar, projeyi açar ve
// projenin çalışma klasörünü döner.
func projeKur(t *testing.T, s *arayuztest.Surucu) string {
	t.Helper()
	s.Yurut(s.Ac("gruplar", grupKur(t, s, "Ekip")), 2)
	s.Tikla("yeniProje")
	s.Yurut(s.FormGonder("projeFormu", map[string]string{"ad": "arayuz"}), 4)
	p := listeOgeleri(t, s.Ekran(), "projeler")
	if len(p) != 1 {
		t.Fatalf("proje kurulamadı: %+v", p)
	}
	s.Yurut(s.Ac("projeler", p[0].ID), 6)
	dir := modeli(s).calismaDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func dosyaYaz(t *testing.T, dir, ad, icerik string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ad), []byte(icerik), 0o644); err != nil {
		t.Fatal(err)
	}
}

// arayuzdenKaydet, Değişiklikler sekmesinden bir kayıt oluşturur.
func arayuzdenKaydet(t *testing.T, s *arayuztest.Surucu, mesaj string) {
	t.Helper()
	s.Yurut(s.SekmeSec("projeSekmeleri", "degisiklikler"), 6)
	s.Yurut(s.FormGonder("commitFormu", map[string]string{"mesaj": mesaj}), 8)
}

func TestArayuz_DegisikliklerVeKayit(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	e := s.Ekran()
	if len(e.Yol) != 3 || e.Yol[2].Metin != "arayuz" {
		t.Errorf("proje kırıntıları = %+v", e.Yol)
	}
	dosyaYaz(t, dir, "a.txt", "merhaba\n")
	s.Yurut(s.SekmeSec("projeSekmeleri", "degisiklikler"), 6)
	d := listeOgeleri(t, s.Ekran(), "degisiklikler")
	if len(d) != 1 || d[0].ID != "a.txt" || len(d[0].Rozetler) == 0 || d[0].Rozetler[0].Metin != "A" {
		t.Fatalf("değişiklikler = %+v", d)
	}
	s.Yurut(s.Sec("degisiklikler", "a.txt"), 6)
	f, ok := arayuztest.Bul(s.Ekran(), "fark")
	if !ok || len(f.Dosyalar) != 1 || !slices.ContainsFunc(f.Dosyalar[0].Satirlar, func(l arayuz.FarkSatiri) bool { return l.Tur == "+" && l.Metin == "merhaba" }) {
		t.Fatalf("fark = %+v", f)
	}
	s.Yurut(s.FormGonder("commitFormu", map[string]string{"mesaj": "ilk"}), 8)
	if !ekrandaVar(s.Ekran(), "Kayıt oluşturuldu") {
		t.Errorf("kayıt iletisi yok: %v", arayuztest.Metinler(s.Ekran()))
	}
	if g, _ := arayuztest.Bul(s.Ekran(), "commitFormu"); g.Cocuklar != nil {
		if m, _ := arayuztest.Bul(s.Ekran(), "mesaj"); m.Deger != "" {
			t.Errorf("kayıttan sonra mesaj kutusu boşalmadı: %q", m.Deger)
		}
	}
	s.Yurut(s.SekmeSec("projeSekmeleri", "gecmis"), 6)
	k := listeOgeleri(t, s.Ekran(), "kayitlar")
	if len(k) != 1 || k[0].Baslik != "ilk" {
		t.Fatalf("geçmiş = %+v", k)
	}
	s.Yurut(s.Sec("kayitlar", k[0].ID), 6)
	if o := listeOgeleri(t, s.Ekran(), "kayitDosyalari"); len(o) != 1 || o[0].ID != "a.txt" {
		t.Errorf("kaydın dosyaları = %+v", o)
	}
	arayuztest.Denetle(t, s.Ekran())
}

// Review Focus 4: işlem sürerken ikinci "Kaydet" yeni bir işlem başlatmaz.
func TestArayuz_CiftKaydet(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	dosyaYaz(t, dir, "a.txt", "x\n")
	s.Yurut(s.SekmeSec("projeSekmeleri", "degisiklikler"), 6)
	if c := s.FormGonder("commitFormu", map[string]string{"mesaj": "bir"}); c == nil {
		t.Fatal("ilk kayıt başlamadı")
	}
	if c := s.FormGonder("commitFormu", map[string]string{"mesaj": "iki"}); c != nil {
		t.Error("işlem sürerken ikinci kayıt başladı")
	}
}

// Review Focus 3: büyük fark protokol sınırını aşmaz. DeltaX taraf başına
// maxDiffSatir (2000) satırdan büyük dosyanın farkını hiç hesaplamaz; iki
// tarafı da sınırın altında olup toplamı aşan fark (1500 satırın hepsi
// değişti) farkSiniri satırda kesilir.
func TestArayuz_BuyukFark(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	var eski, yeni strings.Builder
	for i := range 1500 {
		fmt.Fprintln(&eski, "eski", i)
		fmt.Fprintln(&yeni, "yeni", i)
	}
	dosyaYaz(t, dir, "buyuk.txt", eski.String())
	arayuzdenKaydet(t, s, "ilk")
	dosyaYaz(t, dir, "buyuk.txt", yeni.String())
	s.Yurut(s.SekmeSec("projeSekmeleri", "degisiklikler"), 6)
	s.Yurut(s.Sec("degisiklikler", "buyuk.txt"), 6)
	f, ok := arayuztest.Bul(s.Ekran(), "fark")
	if !ok || len(f.Dosyalar) != 1 {
		t.Fatalf("fark yok: %v", arayuztest.Metinler(s.Ekran()))
	}
	d := f.Dosyalar[0]
	son := d.Satirlar[len(d.Satirlar)-1]
	kalan := len(modeli(s).diffSatirlar) - farkSiniri
	if len(d.Satirlar) != farkSiniri+1 || son.Tur != "@" || kalan <= 0 || !strings.Contains(son.Metin, strconv.Itoa(kalan)) {
		t.Errorf("%d satır (+%d −%d), son satır %+v; %d satır ve '… %d satır daha' bekleniyordu", len(d.Satirlar), d.Eklenen, d.Silinen, son, farkSiniri+1, kalan)
	}
	arayuztest.Denetle(t, s.Ekran())
}

func TestArayuz_DalOlusturVeGec(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	dosyaYaz(t, dir, "a.txt", "x\n")
	arayuzdenKaydet(t, s, "ilk")
	s.Yurut(s.SekmeSec("projeSekmeleri", "dallar"), 4)
	s.Tikla("yeniDal")
	if s.Ekran().Diyalog == nil {
		t.Fatal("yeni dal diyaloğu açılmadı")
	}
	s.FormGonder("dalFormu", map[string]string{"ad": "deneme"})
	if !slices.ContainsFunc(listeOgeleri(t, s.Ekran(), "dallarListesi"), func(o arayuz.ListeOgesi) bool { return o.ID == "deneme" }) {
		t.Fatal("yeni dal listede yok")
	}
	s.TiklaSatir("dalaGec", "deneme")
	if _, ok := arayuztest.Bul(s.Ekran(), "onayEvet"); !ok {
		t.Fatal("dal değiştirme onayı sorulmadı")
	}
	s.Yurut(s.Tikla("onayEvet"), 8)
	if d, ok := arayuztest.Bul(s.Ekran(), "dal"); !ok || d.Deger != "deneme" {
		t.Errorf("etkin dal = %+v", d)
	}
}

func TestArayuz_DalAdiGecersiz(t *testing.T) {
	s := arayuzSurucusu(t)
	projeKur(t, s)
	s.Yurut(s.SekmeSec("projeSekmeleri", "dallar"), 4)
	s.Tikla("yeniDal")
	s.FormGonder("dalFormu", map[string]string{"ad": "a b"})
	if !ekrandaVar(s.Ekran(), "Dal adında boşluk ya da eğik çizgi olamaz.") {
		t.Errorf("geçersiz dal adı uyarısı yok: %v", arayuztest.Metinler(s.Ekran()))
	}
}

// Düzenleyici, dosyayı olduğu gibi kaydeder: 99 satırdan uzun ve sekmeli
// içerik kırpılmaz, sekmeler boşluğa dönmez.
func TestArayuz_DosyaDuzenle(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	s.Yurut(s.SekmeSec("projeSekmeleri", "dosyalar"), 6)
	s.Tikla("yeniDosya")
	if s.Ekran().Diyalog == nil {
		t.Fatal("yeni dosya diyaloğu açılmadı")
	}
	s.Yurut(s.FormGonder("yeniDosyaFormu", map[string]string{"yol": "b.go"}), 6)
	if _, ok := arayuztest.Bul(s.Ekran(), "duzenleyici"); !ok {
		t.Fatalf("düzenleyici açılmadı: %v", arayuztest.Metinler(s.Ekran()))
	}
	icerik := strings.Repeat("func f() {\n\treturn\n}\n", 60) // 180 satır, sekmeli
	s.Yurut(s.FormGonder("duzenleyici", map[string]string{"icerik": icerik}), 6)
	b, err := os.ReadFile(filepath.Join(dir, "b.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != icerik {
		t.Errorf("kaydedilen içerik farklı (%d bayt, beklenen %d)", len(b), len(icerik))
	}
	if d, _ := arayuztest.Bul(s.Ekran(), "icerik"); d.Deger != icerik {
		t.Errorf("düzenleyici kaydedilen içeriği göstermiyor (%d bayt)", len(d.Deger))
	}
	s.Tikla("duzenleyiciKapat")
	if _, ok := arayuztest.Bul(s.Ekran(), "duzenleyici"); ok {
		t.Error("düzenleyici kapanmadı")
	}
	if !slices.ContainsFunc(listeOgeleri(t, s.Ekran(), "dosyalar"), func(o arayuz.ListeOgesi) bool { return o.ID == "b.go" }) {
		t.Error("yeni dosya listede yok")
	}
}

// Review Focus 3: 2 MiB'tan büyük dosya düzenleyicide açılmaz.
func TestArayuz_BuyukDosya(t *testing.T) {
	s := arayuzSurucusu(t)
	dir := projeKur(t, s)
	dosyaYaz(t, dir, "c.txt", strings.Repeat("x", 3<<20))
	s.Yurut(s.SekmeSec("projeSekmeleri", "dosyalar"), 6)
	s.Yurut(s.Ac("dosyalar", "c.txt"), 6)
	if _, ok := arayuztest.Bul(s.Ekran(), "duzenleyici"); ok {
		t.Fatal("3 MiB'lık dosya düzenleyicide açıldı")
	}
	if !ekrandaVar(s.Ekran(), "dosya çok büyük") {
		t.Errorf("büyük dosya iletisi yok: %v", arayuztest.Metinler(s.Ekran()))
	}
	arayuztest.Denetle(t, s.Ekran())
}

// Gelen dosya onayı her görünümün üstünde diyalog olarak açılır.
func TestArayuz_DosyaOnayi(t *testing.T) {
	s := arayuzSurucusu(t)
	s.Gonder(consentPollResultMsg{reqs: []pluginapi.FileConsentReq{{TransferID: "t1", SenderID: pDavetli, Filename: "x.zip", Size: 2048}}})
	if _, ok := arayuztest.Bul(s.Ekran(), "dosyaOnayEvet"); !ok {
		t.Fatalf("dosya onayı diyaloğu yok: %+v", s.Ekran().Diyalog)
	}
	arayuztest.Denetle(t, s.Ekran())
	s.Tikla("dosyaOnayEvet")
	if m := modeli(s); m.pendingConsent != nil || !strings.Contains(m.notice, "kabul edildi") {
		t.Errorf("onay işlenmedi: %q", m.notice)
	}
	if s.Ekran().Diyalog != nil {
		t.Error("onaydan sonra diyalog kapanmadı")
	}
}

// Gönder, çek ve dışa aktar diyalogları; Uzak sekmesi.
func TestArayuz_ProjeDiyaloglari(t *testing.T) {
	s := arayuzSurucusu(t)
	projeKur(t, s)
	for olay, id := range map[string]string{"gonder": "gonderHedefleri", "cek": "cekHedefleri", "disaAktar": "disaAktarFormu"} {
		s.Tikla(olay)
		if _, ok := arayuztest.Bul(s.Ekran(), id); !ok || s.Ekran().Diyalog == nil {
			t.Errorf("%s diyaloğu (%s) açılmadı", olay, id)
		}
		arayuztest.Denetle(t, s.Ekran())
		s.Yurut(s.DiyalogKapat(), 4) // kapanınca son sekme yenilenir
		if s.Ekran().Diyalog != nil {
			t.Errorf("%s diyaloğu kapanmadı", olay)
		}
	}
	s.Yurut(s.SekmeSec("projeSekmeleri", "uzak"), 4)
	if _, ok := arayuztest.Bul(s.Ekran(), "uzaklar"); !ok {
		t.Error("Uzak sekmesinde liste yok")
	}
}

// TestArayuz_HerDilde: arayüz kipindeki her görünüm ve diyalog altı dilde
// çevrilmemiş metin içermez ve AND'ın kabul edeceği bir ekrandır. Durumlar
// terminal ekran testindekilerle aynıdır (aynı model, aynı tuşlar).
func TestArayuz_HerDilde(t *testing.T) {
	denetci := diltest.YeniDenetci(t,
		[]string{"ayşe", "kurucu", "İlk sürüm", "Başlık güncellendi", "Merhaba dünya", "Web Sitesi"},
		diltest.Oku(t, "ceviri.json", ceviriVerisi))
	diltest.HerDilde(t, func(kod string) {
		m, _, _ := ornekProje(t, 140, 40)
		m.arayuzKipi = true
		grup := bas(t, m, "esc")
		ana := bas(t, grup, "esc", "esc")
		ekranlar := map[string]model{
			"proje":         m,
			"degisiklikler": bas(t, m, "d"),
			"gunluk":        bas(t, m, "l"),
			"kayit":         bas(t, m, "l", "enter"),
			"kayit-fark":    bas(t, m, "l", "enter", "enter"),
			"dallar":        bas(t, m, "b"),
			"yeni-dal":      bas(t, m, "b", "n"),
			"dal-sil":       bas(t, m, "b", "down", "d"),
			"uzak":          bas(t, m, "m"),
			"gonder":        bas(t, m, "p"),
			"pull":          bas(t, m, "u"),
			"dosyalar":      bas(t, m, "f"),
			"yeni-dosya":    bas(t, m, "f", "n"),
			"disa-aktar":    bas(t, m, "e"),
			"grup":          grup,
			"davet":         bas(t, grup, "i"),
			"yoneticiler":   bas(t, grup, "y"),
			"grubu-kapat":   bas(t, grup, "x"),
			"ana":           ana,
			"yeni-grup":     bas(t, ana, "down", "enter"),
		}
		for ad, e := range ekranlar {
			ekran := e.Arayuz()
			arayuztest.Denetle(t, ekran)
			for _, k := range denetci.Kalintilar(kod, strings.Join(arayuztest.Metinler(ekran), "\n")) {
				t.Errorf("%s %s: %s", kod, ad, k)
			}
		}
	})
}
