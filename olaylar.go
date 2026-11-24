package main

// olaylar.go — arayüz kipindeki kullanıcı olaylarının işlenmesi. Bir olay önce
// seçimi ya da form değerini modele yazar, sonra terminal arayüzünün tuş
// işleyicisini ilgili tuşla çalıştırır: doğrulamalar ve yan etkiler iki
// arayüzde tektir. Çıkış tuşları (q, ctrl+c, ana menüde esc) arayüzden hiçbir
// zaman gönderilmez: eklentiyi AND kapatır.

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/arayuz"
)

// tusMsg, tuşun adından (enter, esc, ctrl+s ya da tek karakter) bir tuş
// mesajı kurar.
func tusMsg(k string) tea.KeyMsg {
	switch k {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// tus, açık ekranın tuş işleyicisini k tuşuyla çalıştırır.
func (m model) tus(k string) (tea.Model, tea.Cmd) { return m.handleKey(tusMsg(k)) }

// diyalogEkranlari, arayüzde diyalog olarak açılan ekranlardır; Esc (ya da
// diyaloğun kapatılması) terminaldeki esc'nin yaptığını yapar.
var diyalogEkranlari = []ekran{ekGrupOlustur, ekProjeOlustur, ekUyeDavet, ekYoneticiSecim, ekOnay, ekYeniDal}

func (m model) olayIsle(o arayuz.Olay) (tea.Model, tea.Cmd) {
	switch o.Olay {
	case arayuz.OlayDiyalogKapat:
		return m.diyalogKapat()
	case arayuz.OlayGeri:
		return m.geri()
	}
	switch o.ID {
	case "yolAna":
		m.notice, m.ekran = "", ekAnaMenu
		return m, nil
	case "vazgec", "onayHayir":
		return m.diyalogKapat()
	case "onayEvet":
		if m.ekran == ekOnay {
			return m.tus("y")
		}

	// Giriş
	case "yeniGrup":
		m.grupAdIn.SetValue("")
		m.notice, m.ekran = "", ekGrupOlustur
		return m, nil
	case "grupFormu":
		m.ekran = ekGrupOlustur
		m.grupAdIn.SetValue(o.Degerler["ad"])
		return m.tus("enter")
	case "gruplar":
		i := slices.IndexFunc(m.gruplar, func(g Grup) bool { return g.ID == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.grupAramaIn.SetValue("")
		m.grupAramaAktif = false
		m.ekran, m.grupIdx = ekGrupListesi, i
		return m.tus("enter")
	case "davetKabul", "davetRed":
		i := slices.IndexFunc(m.bekleyenDavetler, func(d bekleyenDavet) bool { return d.GrupID == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.ekran, m.davetIdx = ekDavetler, i
		if o.ID == "davetKabul" {
			return m.tus("a")
		}
		return m.tus("r")
	case "sonKayitlar":
		return m.sonKayidaGit(o.Deger)

	// Grup
	case "yeniProje":
		m.projeAdIn.SetValue("")
		m.notice, m.ekran = "", ekProjeOlustur
		return m, nil
	case "projeFormu":
		m.ekran = ekProjeOlustur
		m.projeAdIn.SetValue(o.Degerler["ad"])
		return m.tus("enter")
	case "projeler":
		i := slices.IndexFunc(m.projeler, func(p Proje) bool { return p.ID == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.ekran, m.projeIdx = ekGrupPaneli, i
		return m.tus("enter")
	case "uyeDavet":
		m.ekran = ekGrupPaneli
		return m.tus("i")
	case "davetFormu":
		m.ekran = ekUyeDavet
		m.uyeDavetIn.SetValue(strings.TrimSpace(o.Degerler["peer"]))
		return m.tus("enter")
	case "yoneticiAta":
		m.ekran = ekGrupPaneli
		return m.tus("y")
	case "yoneticiler":
		i := slices.IndexFunc(m.yoneticiHar, func(u Uye) bool { return u.PeerID == o.Deger })
		if i < 0 || m.ekran != ekYoneticiSecim {
			return m, nil
		}
		m.yoneticiIdx = i
		return m.tus("enter")
	case "gruptanCik":
		m.ekran = ekGrupPaneli
		return m.tus("x")
	}
	if m.aktifProje != nil {
		return m.projeOlayi(o)
	}
	return m, nil
}

// projeOlayi, proje görünümünün olaylarıdır.
func (m model) projeOlayi(o arayuz.Olay) (tea.Model, tea.Cmd) {
	switch o.ID {
	case "yolGrup":
		m.notice, m.ekran = "", ekGrupPaneli
		return m, nil
	case "projeSekmeleri":
		for _, s := range projeSekmeleri {
			if s.kod == o.Deger {
				m.notice = ""
				return m.sekmeyiAc(s.ekran)
			}
		}
	case "yenile":
		return m.sekmeyiAc(m.sonSekme)
	case "dal":
		return m.dalDegistirmeyiSor(o.Deger), nil

	// Değişiklikler
	case "degisiklikler", "geriAl":
		liste := degisiklikListesi(m.durum)
		i := slices.IndexFunc(liste, func(d DosyaDegisikligi) bool { return d.Yol == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.ekran, m.durumIdx = ekDurum, i
		if o.ID == "geriAl" {
			return m.tus("g")
		}
		return m.tus("enter")
	case "commitFormu":
		m.ekran = ekCommitMesaj
		m.commitMsgIn.SetValue(o.Degerler["mesaj"])
		return m.tus("enter")

	// Geçmiş
	case "kayitlar":
		i := slices.IndexFunc(m.gunlukGorunur(), func(k LogGirdisi) bool { return k.Hash == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.ekran, m.gunlukIdx = ekGunluk, i
		return m.tus("enter")
	case "kayitDosyalari":
		i := slices.IndexFunc(m.kayit.degisiklikler, func(d DosyaDegisikligi) bool { return d.Yol == o.Deger })
		if i < 0 {
			return m, nil
		}
		m.ekran, m.kayit.idx = ekKayit, i
		return m.tus("enter")
	case "buradanDal":
		m.ekran = ekKayit
		return m.tus("b")

	// Dallar
	case "yeniDal":
		m.ekran = ekDallar
		return m.tus("n")
	case "dalFormu":
		m.ekran = ekYeniDal
		m.yeniDalIn.SetValue(o.Degerler["ad"])
		return m.tus("enter")
	case "dalaGec", "dalSil":
		i := indexOf(m.dallar, o.Deger)
		if i < 0 {
			return m, nil
		}
		m.ekran, m.dalIdx = ekDallar, i
		if o.ID == "dalSil" {
			return m.tus("d")
		}
		return m.tus("enter")
	}
	return m, nil
}

// dalDegistirmeyiSor, araç çubuğundaki dal seçiminden sonra dal
// değiştirmenin onayını açar (terminaldeki Dallar ekranındaki enter gibi).
func (m model) dalDegistirmeyiSor(hedef string) model {
	if m.pending || hedef == m.aktifProje.AktifDal {
		return m
	}
	m.dallar = dalAnahtarlari(m.aktifProje.Dallar)
	i := indexOf(m.dallar, hedef)
	if i < 0 {
		return m
	}
	m.dalIdx = i
	m.onayTip, m.onayGeriEkran = onayDalDegistir, m.ekran
	if slices.Contains(diyalogEkranlari, m.onayGeriEkran) {
		m.onayGeriEkran = ekProjePaneli
	}
	m.ekran = ekOnay
	return m
}

// diyalogKapat, açık diyaloğu terminaldeki esc gibi kapatır.
func (m model) diyalogKapat() (tea.Model, tea.Cmd) {
	if slices.Contains(diyalogEkranlari, m.ekran) {
		return m.tus("esc")
	}
	return m, nil
}

// geri, bir üst görünüme döner (diyalog açıksa onu kapatır). Girişte bir
// şey yapmaz: eklentiyi AND kapatır.
func (m model) geri() (tea.Model, tea.Cmd) {
	if slices.Contains(diyalogEkranlari, m.ekran) {
		return m.diyalogKapat()
	}
	switch arka := m.arkaPlanEkrani(); {
	case arka == ekGrupPaneli:
		m.notice, m.ekran = "", ekAnaMenu
	case m.ekran == ekDiff || m.ekran == ekKayit:
		return m.tus("esc") // fark → listesi, kayıt → geçmiş
	case projeEkraniMi(arka):
		m.notice, m.ekran = "", ekGrupPaneli
	}
	return m, nil
}

// sonKayidaGit, girişteki "son kayıtlar"dan birine (grupID/projeID/hash)
// gider: grubu ve projeyi açar.
func (m model) sonKayidaGit(anahtar string) (tea.Model, tea.Cmd) {
	parca := strings.SplitN(anahtar, "/", 3)
	if len(parca) != 3 {
		return m, nil
	}
	gi := slices.IndexFunc(m.gruplar, func(g Grup) bool { return g.ID == parca[0] })
	if gi < 0 {
		return m, nil
	}
	g := m.gruplar[gi]
	m.aktifGrup = &g
	m.projeler = projeleriListele(m.dataDir, g.ID)
	pi := slices.IndexFunc(m.projeler, func(p Proje) bool { return p.ID == parca[1] })
	if pi < 0 {
		m.ekran = ekGrupPaneli
		return m, peersPollCmd(m.client)
	}
	m.ekran, m.projeIdx = ekGrupPaneli, pi
	return m.tus("enter")
}
