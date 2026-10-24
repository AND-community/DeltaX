package main

// inceleme.go — geçmişi ve değişiklikleri inceleme yardımcıları: bir
// kaydın değiştirdiği dosyalar ve farkları, çalışma klasöründeki bir
// değişikliği geri alma, bir üyenin ucunda yerelde olmayan kayıtların
// sayısı, uzun farkların katlanması ve göreli zaman.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lucian95511/and/dil"
)

// DosyaDegisikligi, bir kaydın (ya da çalışma klasörünün) değiştirdiği tek
// bir dosyadır. Tip: '+' eklendi, '~' değişti, '-' silindi.
type DosyaDegisikligi struct {
	Yol string
	Tip byte
}

// agacFarki, iki ağaç arasında değişen dosyaları yola göre sıralı döndürür.
func agacFarki(eski, yeni map[string]AgacGirdi) []DosyaDegisikligi {
	var out []DosyaDegisikligi
	for yol, g := range yeni {
		if e, ok := eski[yol]; !ok {
			out = append(out, DosyaDegisikligi{yol, '+'})
		} else if e.Hash != g.Hash {
			out = append(out, DosyaDegisikligi{yol, '~'})
		}
	}
	for yol := range eski {
		if _, ok := yeni[yol]; !ok {
			out = append(out, DosyaDegisikligi{yol, '-'})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Yol < out[j].Yol })
	return out
}

// kayitAgaclari, kaydın ve ilk ebeveyninin ağaçlarını döndürür (ilk kayıtta
// ebeveyn ağacı boştur).
func kayitAgaclari(dataDir, projeID, hash string) (c Commit, ebeveyn, kendi map[string]AgacGirdi, err error) {
	c, err = commitOku(dataDir, projeID, hash)
	if err != nil {
		return c, nil, nil, err
	}
	kendi, err = agacHaritala(dataDir, projeID, c.Agac)
	if err != nil {
		return c, nil, nil, err
	}
	ebeveyn = map[string]AgacGirdi{}
	if len(c.Ebeveynler) > 0 {
		pc, perr := commitOku(dataDir, projeID, c.Ebeveynler[0])
		if perr != nil {
			return c, nil, nil, fmt.Errorf(dil.T("önceki kayıt %s okunamadı: %w"), kisaHash(c.Ebeveynler[0]), perr)
		}
		if ebeveyn, err = agacHaritala(dataDir, projeID, pc.Agac); err != nil {
			return c, nil, nil, err
		}
	}
	return c, ebeveyn, kendi, nil
}

// kayitDegisiklikleri, bir kaydın ilk ebeveynine göre değiştirdiği dosyalardır.
func kayitDegisiklikleri(dataDir, projeID, hash string) (Commit, []DosyaDegisikligi, error) {
	c, ebeveyn, kendi, err := kayitAgaclari(dataDir, projeID, hash)
	if err != nil {
		return c, nil, err
	}
	return c, agacFarki(ebeveyn, kendi), nil
}

// kayitDosyaFarki, bir kayıtta bir dosyanın ilk ebeveyne göre satır farkıdır.
func kayitDosyaFarki(dataDir, projeID, hash, yol string) ([]DiffSatiri, error) {
	_, ebeveyn, kendi, err := kayitAgaclari(dataDir, projeID, hash)
	if err != nil {
		return nil, err
	}
	oku := func(g AgacGirdi, ok bool) ([]byte, error) {
		if !ok {
			return nil, nil
		}
		if g.Boyut > maxDiffBoyutu {
			return nil, fmt.Errorf(dil.T("dosya çok büyük — satır bazlı fark atlandı: %s"), yol)
		}
		_, data, err := nesneOku(dataDir, projeID, g.Hash)
		return data, err
	}
	eskiG, eskiOK := ebeveyn[yol]
	yeniG, yeniOK := kendi[yol]
	eski, err := oku(eskiG, eskiOK)
	if err != nil {
		return nil, err
	}
	yeni, err := oku(yeniG, yeniOK)
	if err != nil {
		return nil, err
	}
	if satirSayisi(eski) > maxDiffSatir || satirSayisi(yeni) > maxDiffSatir {
		return nil, fmt.Errorf(dil.T("dosya çok fazla satır içeriyor — satır bazlı fark atlandı: %s"), yol)
	}
	return satirDiff(eski, yeni), nil
}

// degisikligiGeriAl, çalışma klasöründeki bir dosyayı son kayıttaki hâline
// döndürür; son kayıtta yoksa (yeni eklenmişse) siler.
func degisikligiGeriAl(dataDir, projeID, calismaDir, headTree, yol string) error {
	if !gecerliGoreliYol(yol) {
		return fmt.Errorf(dil.T("güvensiz dosya yolu reddedildi: %q"), yol)
	}
	head, err := agacHaritala(dataDir, projeID, headTree)
	if err != nil {
		return err
	}
	if g, ok := head[yol]; ok {
		return dosyayiCalismayaYaz(dataDir, projeID, calismaDir, g)
	}
	full := filepath.Join(calismaDir, filepath.FromSlash(yol))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	bosDizinleriTemizle(calismaDir, filepath.Dir(full))
	return nil
}

// bosDizinleriTemizle, dizin'den başlayıp çalışma klasörünün köküne kadar
// boşalan üst dizinleri siler.
func bosDizinleriTemizle(kok, dizin string) {
	kok = filepath.Clean(kok)
	for d := filepath.Clean(dizin); d != kok && len(d) > len(kok); d = filepath.Dir(d) {
		if os.Remove(d) != nil {
			return // boş değil ya da silinemedi
		}
	}
}

// uzakYeniKayitSayisi, bir üyenin ucunda olup yerel dalda bulunmayan kayıt
// sayısıdır: 0 ise o uçtaki her şey zaten birleştirilmiştir.
func uzakYeniKayitSayisi(dataDir, projeID, yerelHead, uzakHead string) int {
	if uzakHead == "" || uzakHead == yerelHead {
		return 0
	}
	return yeniKayitSay(dataDir, projeID, atalariTopla(dataDir, projeID, yerelHead), uzakHead)
}

// yeniKayitSay, uzakHead'den ulaşılan ve yerel kümesinde olmayan kayıt
// sayısıdır; yerel küme birçok uç için bir kez hesaplanır.
func yeniKayitSay(dataDir, projeID string, yerel map[string]bool, uzakHead string) int {
	if uzakHead == "" || yerel[uzakHead] {
		return 0
	}
	n := 0
	for h := range atalariTopla(dataDir, projeID, uzakHead) {
		if !yerel[h] {
			n++
		}
	}
	return n
}

// diffBaglam, katlanmış farkta bir değişikliğin önünde ve ardında
// gösterilen değişmemiş satır sayısıdır.
const diffBaglam = 3

// diffKatla, değişikliklerden uzaktaki değişmemiş satırları tek bir
// "⋯ N satır aynı" satırına indirir (Tip '@'). Uzun bir dosyada tek satırlık
// bir değişikliği bulmak için tüm dosyayı kaydırmak gerekmez.
func diffKatla(satirlar []DiffSatiri, baglam int) []DiffSatiri {
	yakin := make([]bool, len(satirlar))
	for i, s := range satirlar {
		if s.Tip == ' ' {
			continue
		}
		for j := max(0, i-baglam); j <= min(len(satirlar)-1, i+baglam); j++ {
			yakin[j] = true
		}
	}
	var out []DiffSatiri
	for i := 0; i < len(satirlar); {
		if yakin[i] {
			out = append(out, satirlar[i])
			i++
			continue
		}
		j := i
		for j < len(satirlar) && !yakin[j] {
			j++
		}
		out = append(out, DiffSatiri{Tip: '@', Metin: dil.F("⋯ %d satır aynı", j-i)})
		i = j
	}
	return out
}

// diffSay, farktaki eklenen ve silinen satır sayılarıdır.
func diffSay(satirlar []DiffSatiri) (eklenen, silinen int) {
	for _, s := range satirlar {
		switch s.Tip {
		case '+':
			eklenen++
		case '-':
			silinen++
		}
	}
	return eklenen, silinen
}

// simdi, görünümlerin kullandığı saattir (testler sabitleyebilsin diye değişken).
var simdi = time.Now

// goreliZaman, t'yi simdi'ye göre kısa ve okunur biçimde yazar.
func goreliZaman(t, simdi time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := simdi.Sub(t)
	switch {
	case d < time.Minute:
		return dil.T("az önce")
	case d < time.Hour:
		return dil.F("%d dk önce", int(d.Minutes()))
	case d < 24*time.Hour:
		return dil.F("%d sa önce", int(d.Hours()))
	case d < 48*time.Hour:
		return dil.T("dün")
	case d < 7*24*time.Hour:
		return dil.F("%d gün önce", int(d.Hours()/24))
	}
	return t.Local().Format("02.01.2006")
}
