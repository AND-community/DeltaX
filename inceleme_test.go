package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// kaydet, çalışma klasörünün o anki hâlini bir kayıt olarak yazar.
func kaydet(t *testing.T, dataDir, projeID, calismaDir, head, mesaj string) string {
	t.Helper()
	agac, _, _, err := agacOlustur(dataDir, projeID, calismaDir)
	if err != nil {
		t.Fatal(err)
	}
	var ebeveyn []string
	if head != "" {
		ebeveyn = []string{head}
	}
	h, err := commitOlustur(dataDir, projeID, agac, ebeveyn, "test", mesaj)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestKayitDegisiklikleriVeFarki(t *testing.T) {
	dataDir, projeID, calisma := yeniTestProjesi(t)
	yazDosya(t, calisma, "a.txt", "bir\niki\n")
	yazDosya(t, calisma, "sil.txt", "x")
	ilk := kaydet(t, dataDir, projeID, calisma, "", "ilk")

	_, degisen, err := kayitDegisiklikleri(dataDir, projeID, ilk)
	if err != nil || len(degisen) != 2 || degisen[0] != (DosyaDegisikligi{"a.txt", '+'}) {
		t.Fatalf("ilk kayıtta her dosya eklenmiş sayılmalı: %+v %v", degisen, err)
	}

	yazDosya(t, calisma, "a.txt", "bir\nüç\n")
	yazDosya(t, calisma, "alt/yeni.txt", "y")
	if err := os.Remove(filepath.Join(calisma, "sil.txt")); err != nil {
		t.Fatal(err)
	}
	ikinci := kaydet(t, dataDir, projeID, calisma, ilk, "ikinci")
	c, degisen, err := kayitDegisiklikleri(dataDir, projeID, ikinci)
	if err != nil || c.Mesaj != "ikinci" {
		t.Fatal(c, err)
	}
	beklenen := []DosyaDegisikligi{{"a.txt", '~'}, {"alt/yeni.txt", '+'}, {"sil.txt", '-'}}
	if len(degisen) != 3 || degisen[0] != beklenen[0] || degisen[1] != beklenen[1] || degisen[2] != beklenen[2] {
		t.Fatalf("değişiklikler: %+v", degisen)
	}

	fark, err := kayitDosyaFarki(dataDir, projeID, ikinci, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if e, s := diffSay(fark); e != 1 || s != 1 {
		t.Fatalf("a.txt: 1 eklenen 1 silinen satır olmalı: %+v", fark)
	}
	fark, _ = kayitDosyaFarki(dataDir, projeID, ikinci, "sil.txt")
	if e, s := diffSay(fark); e != 0 || s != 1 {
		t.Fatalf("silinen dosyanın farkı yalnızca silinen satırlar: %+v", fark)
	}
}

func TestDegisikligiGeriAl(t *testing.T) {
	dataDir, projeID, calisma := yeniTestProjesi(t)
	yazDosya(t, calisma, "a.txt", "orijinal")
	yazDosya(t, calisma, "b.txt", "silinecek")
	head := kaydet(t, dataDir, projeID, calisma, "", "ilk")
	tree, _ := headTreeOf(dataDir, projeID, head)

	yazDosya(t, calisma, "a.txt", "bozuldu")
	if err := os.Remove(filepath.Join(calisma, "b.txt")); err != nil {
		t.Fatal(err)
	}
	yazDosya(t, calisma, "yeni/klasor/c.txt", "yeni")

	for _, yol := range []string{"a.txt", "b.txt", "yeni/klasor/c.txt"} {
		if err := degisikligiGeriAl(dataDir, projeID, calisma, tree, yol); err != nil {
			t.Fatalf("%s: %v", yol, err)
		}
	}
	d, err := durumHesapla(dataDir, projeID, calisma, tree)
	if err != nil || !d.temizMi() {
		t.Fatalf("geri almadan sonra çalışma klasörü temiz olmalı: %+v %v", d, err)
	}
	if _, err := os.Stat(filepath.Join(calisma, "yeni")); !os.IsNotExist(err) {
		t.Fatal("yeni dosyayla boşalan klasörler de silinmeli")
	}
	if err := degisikligiGeriAl(dataDir, projeID, calisma, tree, "../disari.txt"); err == nil {
		t.Fatal("çalışma klasörü dışına çıkan yol reddedilmeli")
	}
}

func TestUzakYeniKayitSayisi(t *testing.T) {
	dataDir, projeID, calisma := yeniTestProjesi(t)
	yazDosya(t, calisma, "a.txt", "1")
	k1 := kaydet(t, dataDir, projeID, calisma, "", "1")
	yazDosya(t, calisma, "a.txt", "2")
	k2 := kaydet(t, dataDir, projeID, calisma, k1, "2")
	yazDosya(t, calisma, "a.txt", "3")
	k3 := kaydet(t, dataDir, projeID, calisma, k2, "3")

	if n := uzakYeniKayitSayisi(dataDir, projeID, k1, k3); n != 2 {
		t.Fatalf("uzak uç 2 kayıt önde olmalı: %d", n)
	}
	if n := uzakYeniKayitSayisi(dataDir, projeID, k3, k1); n != 0 {
		t.Fatalf("yerelin atası olan uç güncel sayılmalı: %d", n)
	}
	if n := uzakYeniKayitSayisi(dataDir, projeID, "", k2); n != 2 {
		t.Fatalf("yerelde kayıt yokken uzaktaki her kayıt yenidir: %d", n)
	}
}

func TestDiffKatla(t *testing.T) {
	var satirlar []DiffSatiri
	for i := 0; i < 20; i++ {
		satirlar = append(satirlar, DiffSatiri{' ', "aynı"})
	}
	satirlar[10] = DiffSatiri{'+', "yeni"}
	k := diffKatla(satirlar, 3)
	var metin []string
	for _, s := range k {
		metin = append(metin, string(s.Tip)+s.Metin)
	}
	got := strings.Join(metin, "|")
	want := "@⋯ 7 satır aynı| aynı| aynı| aynı|+yeni| aynı| aynı| aynı|@⋯ 6 satır aynı"
	if got != want {
		t.Fatalf("katlanmış fark:\n got %s\nwant %s", got, want)
	}
	if len(diffKatla(nil, 3)) != 0 {
		t.Fatal("boş fark boş kalmalı")
	}
}

func TestGoreliZaman(t *testing.T) {
	simdi := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for d, beklenen := range map[time.Duration]string{
		10 * time.Second:   "az önce",
		5 * time.Minute:    "5 dk önce",
		3 * time.Hour:      "3 sa önce",
		30 * time.Hour:     "dün",
		4 * 24 * time.Hour: "4 gün önce",
	} {
		if got := goreliZaman(simdi.Add(-d), simdi); got != beklenen {
			t.Errorf("%v: %q; %q beklenirdi", d, got, beklenen)
		}
	}
	if goreliZaman(time.Time{}, simdi) != "" {
		t.Error("sıfır zaman boş yazılmalı")
	}
}
