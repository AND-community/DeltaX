package main

// arayuz_proje.go — arayüz kipinde proje görünümü: araç çubuğu (dal, gönder,
// çek, dışa aktar, klasör), özet satırı ve sekmeler (Değişiklikler, Geçmiş,
// Dallar, Uzak, Dosyalar). Sekmeler terminaldeki proje ekranlarının
// karşılığıdır; hangisinin açık olduğu m.ekran'dan çıkar.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/arayuz"
	"github.com/lucian95511/and/dil"
)

// farkSiniri, bir dosyanın farkından arayüze gönderilen en çok satırdır
// (protokol sınırları aşılmasın; kalanı tek satırla belirtilir).
const farkSiniri = 2000

// Proje sekmeleri: arayüzdeki kimliği, terminaldeki ekranı ve o ekranı açan
// proje paneli tuşu.
var projeSekmeleri = []struct {
	kod   string
	ekran ekran
	tus   string
}{
	{"degisiklikler", ekDurum, "d"},
	{"gecmis", ekGunluk, "l"},
	{"dallar", ekDallar, "b"},
	{"uzak", ekUzakDallar, "m"},
	{"dosyalar", ekDosyaListesi, "f"},
}

// sekmesi, proje görünümünde açık olan sekmenin kodudur.
func (m model) sekmesi(e ekran) string {
	switch e {
	case ekDurum, ekCommitMesaj:
		return "degisiklikler"
	case ekGunluk, ekKayit:
		return "gecmis"
	case ekDiff:
		if m.diffGeri == ekKayit {
			return "gecmis"
		}
		return "degisiklikler"
	case ekYeniDal:
		if m.dalKaynak != "" {
			return "gecmis"
		}
		return "dallar"
	case ekDallar:
		return "dallar"
	case ekUzakDallar:
		return "uzak"
	case ekDosyaListesi, ekYeniDosya, ekDosyaDuzenle:
		return "dosyalar"
	}
	for _, s := range projeSekmeleri {
		if s.ekran == m.sonSekme {
			return s.kod
		}
	}
	return "degisiklikler"
}

// sekmeyiAc, proje görünümünde e ekranının sekmesini açar (terminaldeki
// proje paneli işlemiyle). Açık ileti korunur.
func (m model) sekmeyiAc(e ekran) (tea.Model, tea.Cmd) {
	tus := "d"
	m.sonSekme = ekDurum
	for _, s := range projeSekmeleri {
		if s.ekran == e {
			tus, m.sonSekme = s.tus, e
		}
	}
	notice, isErr := m.notice, m.isErr
	m.ekran = ekProjePaneli
	yeni, cmd := m.projeIsleminiCalistir(tus)
	if y, ok := yeni.(model); ok {
		if y.notice == "" {
			y.notice, y.isErr = notice, isErr
		}
		return y, cmd
	}
	return yeni, cmd
}

func degisiklikHarfi(tip byte) (string, string) {
	switch tip {
	case '+':
		return "A", "iyi"
	case '-':
		return "D", "hata"
	}
	return "M", "vurgu"
}

// farkOgesi, bir dosyanın farkıdır (satır numaralarıyla, en çok farkSiniri
// satır).
func farkOgesi(satirlar []DiffSatiri, yol, durum string) arayuz.Oge {
	f := arayuz.FarkDosyasi{Yol: safe(yol), Durum: durum}
	eski, yeni := 0, 0
	for i, s := range satirlar {
		l := arayuz.FarkSatiri{Tur: string(s.Tip), Metin: s.Metin}
		switch s.Tip {
		case '+':
			yeni++
			f.Eklenen++
			l.Yeni = yeni
		case '-':
			eski++
			f.Silinen++
			l.Eski = eski
		default:
			eski++
			yeni++
			l.Tur, l.Eski, l.Yeni = " ", eski, yeni
		}
		if i < farkSiniri {
			f.Satirlar = append(f.Satirlar, l)
		}
	}
	if n := len(satirlar) - farkSiniri; n > 0 {
		f.Satirlar = append(f.Satirlar, arayuz.FarkSatiri{Tur: "@", Metin: dil.F("… %d satır daha", n)})
	}
	o := arayuz.Fark(f)
	o.ID = "fark"
	return o
}

// projeAraclari, proje görünümünün araç çubuğudur.
func (m model) projeAraclari() []arayuz.Oge {
	p := m.aktifProje
	var araclar []arayuz.Oge
	if dallar := dalAnahtarlari(p.Dallar); len(dallar) > 0 {
		var secenekler []arayuz.Secenek
		for _, d := range dallar {
			secenekler = append(secenekler, arayuz.Secenek{Deger: d, Etiket: "⎇ " + safe(d)})
		}
		araclar = append(araclar, arayuz.Secim("dal", "", p.AktifDal, secenekler, arayuz.Aninda, arayuz.DevreDisiIse(m.pending)))
	}
	return append(araclar,
		arayuz.Dugme("gonder", dil.T("Gönder"), arayuz.IleIkon("gonder"), arayuz.DevreDisiIse(m.pending)),
		arayuz.Dugme("cek", dil.T("Çek"), arayuz.IleIkon("asagi"), arayuz.DevreDisiIse(m.pending)),
		arayuz.Dugme("disaAktar", dil.T("Dışa aktar"), arayuz.Hayalet, arayuz.IleIkon("belge")),
		arayuz.Dugme("klasoruAc", dil.T("Klasörü aç"), arayuz.Hayalet, arayuz.IleIkon("dis")),
	)
}

// ozetSatiri, projenin üst satırıdır: etkin dal, son kayıt, bekleyen
// değişiklikler ve uzaklardaki yeni kayıtlar.
func (m model) ozetSatiri() arayuz.Oge {
	p := m.aktifProje
	parcalar := []arayuz.Parca{{Metin: "⎇ " + safe(p.AktifDal), Stil: "kod"}}
	if o := m.ozet; o != nil && o.projeID == p.ID {
		if o.son != nil {
			parcalar = append(parcalar, arayuz.Parca{Metin: ilkSatirSafe(o.son.Commit.Mesaj)},
				arayuz.Parca{Metin: safe(o.son.Commit.Yazar) + " · " + o.son.Commit.TS.Format(zamanBicimi), Stil: "soluk"})
		} else {
			parcalar = append(parcalar, arayuz.Parca{Metin: dil.T("Henüz kayıt yok."), Stil: "soluk"})
		}
		if n := len(degisiklikListesi(o.durum)); n > 0 {
			parcalar = append(parcalar, arayuz.Parca{Metin: dil.F("%d değişiklik", n), Stil: "uyari"})
		}
		if n := toplamYeni(o.uzakYeni); n > 0 {
			parcalar = append(parcalar, arayuz.Parca{Metin: dil.F("uzaklarda %d yeni kayıt", n), Stil: "iyi"})
		}
	}
	return arayuz.Zengin(parcalar...)
}

// projeGorunumu, proje görünümünün gövdesidir; arka, açık sekmenin ekranıdır.
func (m model) projeGorunumu(arka ekran) arayuz.Oge {
	kod := m.sekmesi(arka)
	sekmeler := []arayuz.Sekme{
		{ID: "degisiklikler", Etiket: dil.T("Değişiklikler"), Ikon: "belge"},
		{ID: "gecmis", Etiket: dil.T("Geçmiş"), Ikon: "kod"},
		{ID: "dallar", Etiket: dil.T("Dallar"), Ikon: "ag"},
		{ID: "uzak", Etiket: dil.T("Uzak"), Ikon: "kure"},
		{ID: "dosyalar", Etiket: dil.T("Dosyalar"), Ikon: "belge"},
	}
	if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID {
		sekmeler[0].Sayac = len(degisiklikListesi(m.ozet.durum))
	}
	var icerik arayuz.Oge
	switch kod {
	case "gecmis":
		icerik = m.gecmisSekmesi(arka)
	case "dallar":
		icerik = m.dallarSekmesi()
	case "uzak":
		icerik = m.uzakSekmesi()
	case "dosyalar":
		icerik = m.dosyalarSekmesi(arka)
	default:
		icerik = m.degisikliklerSekmesi(arka)
	}
	return arayuz.Yigin(m.ozetSatiri(), arayuz.SekmeCubugu("projeSekmeleri", kod, sekmeler...), icerik)
}

func (m model) degisikliklerSekmesi(arka ekran) arayuz.Oge {
	liste := degisiklikListesi(m.durum)
	secili := ""
	if arka == ekDiff && m.diffGeri == ekDurum {
		secili = m.diffBaslik
	}
	var ogeler []arayuz.ListeOgesi
	for _, d := range liste {
		harf, renk := degisiklikHarfi(d.Tip)
		ogeler = append(ogeler, arayuz.ListeOgesi{ID: d.Yol, Baslik: safe(d.Yol), Ikon: "belge",
			Rozetler: []arayuz.Rozet{{Metin: harf, Renk: renk}},
			Dugmeler: []arayuz.Oge{arayuz.Dugme("geriAl", dil.T("Geri al"), arayuz.Hayalet)}})
	}
	form := arayuz.Form("commitFormu",
		arayuz.Girdi("mesaj", dil.T("Kayıt mesajı"), m.commitMsgIn.Value(), arayuz.IleEnUzun(200),
			arayuz.IleYerTutucu(dil.T("Bu kayıtta ne değişti?"))),
		arayuz.SatirOge(
			arayuz.Dugme("kaydet", dil.T("Kaydet"), arayuz.Birincil, arayuz.Gonderir, arayuz.IleIkon("tamam"),
				arayuz.IleKisayol("ctrl+enter"), arayuz.DevreDisiIse(len(liste) == 0 || m.pending)),
			arayuz.Dugme("yenile", dil.T("Yenile"), arayuz.Hayalet, arayuz.IleIkon("yenile"), arayuz.DevreDisiIse(m.pending)),
		))
	sol := arayuz.Yigin(
		arayuz.Liste("degisiklikler", ogeler, arayuz.IleSecili(secili), arayuz.IleBos(dil.T("Kaydedilecek değişiklik yok."))),
		form,
	)
	var sag arayuz.Oge
	if secili != "" {
		durum := "M"
		for _, d := range liste {
			if d.Yol == secili {
				durum, _ = degisiklikHarfi(d.Tip)
			}
		}
		sag = farkOgesi(m.diffSatirlar, secili, durum)
	} else {
		sag = arayuz.Bos("belge", dil.T("Bir dosya seç"), dil.T("Seçtiğin dosyanın son kayda göre farkı burada görünür."))
	}
	return arayuz.Bolunmus(sol, sag)
}

func (m model) gecmisSekmesi(arka ekran) arayuz.Oge {
	kayitAcik := arka == ekKayit || (arka == ekDiff && m.diffGeri == ekKayit) || (arka == ekYeniDal && m.dalKaynak != "")
	secili := ""
	if kayitAcik {
		secili = m.kayit.hash
	}
	var ogeler []arayuz.ListeOgesi
	for _, k := range m.gunlukGorunur() {
		ogeler = append(ogeler, arayuz.ListeOgesi{ID: k.Hash, Baslik: ilkSatirSafe(k.Commit.Mesaj), Ikon: "kod",
			Alt: safe(k.Commit.Yazar) + " · " + kisa7(k.Hash), Sag: k.Commit.TS.Format(zamanBicimi)})
	}
	sol := arayuz.Liste("kayitlar", ogeler, arayuz.IleSecili(secili), arayuz.Aranabilir, arayuz.IleBos(dil.T("Henüz kayıt yok.")))
	if !kayitAcik {
		return arayuz.Bolunmus(sol, arayuz.Bos("kod", dil.T("Bir kayıt seç"), dil.T("Seçtiğin kaydın ayrıntısı ve değişen dosyaları burada görünür.")))
	}
	k := m.kayit
	mesaj := safe(strings.TrimSpace(k.commit.Mesaj))
	baslik, govde, _ := strings.Cut(mesaj, "\n")
	kart := arayuz.Kart(arayuz.Baslik(3, baslik))
	if govde = strings.TrimSpace(govde); govde != "" {
		kart.Cocuklar = append(kart.Cocuklar, arayuz.Metin(govde))
	}
	kart.Cocuklar = append(kart.Cocuklar, arayuz.AnahtarDeger(
		arayuz.Cift{Anahtar: dil.T("Kayıt"), Deger: k.hash, Kod: true, Kopyala: true},
		arayuz.Cift{Anahtar: dil.T("Yazar"), Deger: safe(k.commit.Yazar)},
		arayuz.Cift{Anahtar: dil.T("Tarih"), Deger: k.commit.TS.Format(zamanBicimi)},
	))
	dosyaSecili := ""
	if arka == ekDiff {
		dosyaSecili = m.diffBaslik
	}
	var dosyalar []arayuz.ListeOgesi
	durum := "M"
	for _, d := range k.degisiklikler {
		harf, renk := degisiklikHarfi(d.Tip)
		if d.Yol == dosyaSecili {
			durum = harf
		}
		dosyalar = append(dosyalar, arayuz.ListeOgesi{ID: d.Yol, Baslik: safe(d.Yol), Ikon: "belge", Rozetler: []arayuz.Rozet{{Metin: harf, Renk: renk}}})
	}
	sag := arayuz.Yigin(kart, bolum(dil.T("Değişen dosyalar"),
		[]arayuz.Oge{arayuz.Dugme("buradanDal", dil.T("Buradan dal oluştur"), arayuz.Hayalet, arayuz.IleIkon("arti"))},
		arayuz.Liste("kayitDosyalari", dosyalar, arayuz.IleSecili(dosyaSecili))))
	if dosyaSecili != "" {
		sag.Cocuklar = append(sag.Cocuklar, farkOgesi(m.diffSatirlar, dosyaSecili, durum))
	}
	return arayuz.Bolunmus(sol, sag)
}

func (m model) dallarSekmesi() arayuz.Oge {
	var ogeler []arayuz.ListeOgesi
	for _, d := range m.dallar {
		o := arayuz.ListeOgesi{ID: d, Baslik: "⎇ " + safe(d), Alt: dil.T("Henüz kayıt yok.")}
		if k, ok := m.dalSon[d]; ok {
			o.Alt, o.Sag = ilkSatirSafe(k.Commit.Mesaj), k.Commit.TS.Format(zamanBicimi)
		}
		if d == m.aktifProje.AktifDal {
			o.Rozetler = []arayuz.Rozet{{Metin: dil.T("etkin"), Renk: "iyi"}}
		} else {
			o.Dugmeler = []arayuz.Oge{arayuz.Dugme("dalaGec", dil.T("Geç")), arayuz.Dugme("dalSil", dil.T("Sil"), arayuz.Tehlikeli)}
		}
		ogeler = append(ogeler, o)
	}
	return bolum(dil.T("Dallar"), []arayuz.Oge{arayuz.Dugme("yeniDal", dil.T("Yeni dal"), arayuz.Hayalet, arayuz.IleIkon("arti"))},
		arayuz.Liste("dallarListesi", ogeler, arayuz.IleBos(dil.T("Henüz kayıt yok."))))
}

// uzakSekmesi ve dosyalarSekmesi arayuz_diyalog.go'da (Görev 10) dolar.
func (m model) uzakSekmesi() arayuz.Oge { return arayuz.Bos("kure", dil.T("Uzak"), "") }

func (m model) dosyalarSekmesi(ekran) arayuz.Oge { return arayuz.Bos("belge", dil.T("Dosyalar"), "") }
