package main

// arayuz.go — DeltaX'in AND masaüstündeki ekranı (arayüz kipi, bkz. paket
// arayuz). Terminal görünümü (view.go) gibi bu da modelin durumundan çizilir:
// m.ekran iki arayüzün ortak kaynağıdır. Diyalog ekranları (yeni grup, onay…)
// altlarındaki görünümün (arkaPlanEkrani) üstünde diyalog olarak açılır.
// Kullanıcının olayları olaylar.go'da tuş işleyicilerine çevrilir.

import (
	"strconv"
	"strings"

	"github.com/lucian95511/and/arayuz"
	"github.com/lucian95511/and/dil"
)

// zamanBicimi, listelerdeki kısa tarih biçimidir (dilden bağımsız).
const zamanBicimi = "02.01.2006 15:04"

// arkaPlanEkrani, açık ekranın altında çizilecek görünümün ekranıdır:
// diyalog ekranları açıldıkları görünüme, onaylar ve yardım geldikleri
// ekrana bağlanır.
func (m model) arkaPlanEkrani() ekran {
	e := m.ekran
	for range 4 {
		switch e {
		case ekOnay:
			e = m.onayGeriEkran
			continue
		case ekDosyaOnay:
			e = m.onayOnceki
			continue
		case ekYardim:
			e = m.yardimGeriEkran
			continue
		}
		break
	}
	switch e {
	case ekAnaMenu, ekKimlik, ekGrupListesi, ekDavetler, ekGrupOlustur:
		return ekAnaMenu
	case ekGrupPaneli, ekProjeOlustur, ekUyeDavet, ekYoneticiSecim:
		return ekGrupPaneli
	}
	return e
}

// Arayuz, DeltaX'in arayüz kipindeki ekranıdır (bkz. arayuz.Model).
func (m model) Arayuz() arayuz.Ekran {
	e := arayuz.Ekran{Ikon: "kod", Mesgul: m.pending}
	var govde arayuz.Oge
	switch arka := m.arkaPlanEkrani(); {
	case projeEkraniMi(arka) && m.aktifProje != nil && m.aktifGrup != nil:
		e.Yol = []arayuz.Kirinti{{ID: "yolAna", Metin: "DeltaX"}, {ID: "yolGrup", Metin: safe(m.aktifGrup.Ad)}, {Metin: safe(m.aktifProje.Ad)}}
		e.Araclar = m.projeAraclari()
		govde = m.projeGorunumu(arka)
		e.Durum = m.uyeDurumu()
	case arka == ekGrupPaneli && m.aktifGrup != nil:
		e.Yol = []arayuz.Kirinti{{ID: "yolAna", Metin: "DeltaX"}, {Metin: safe(m.aktifGrup.Ad)}}
		sol, sag := m.grupGorunumu()
		govde = arayuz.Bolunmus(arayuz.Yigin(sol...), arayuz.Yigin(sag...))
		e.Durum = m.uyeDurumu()
	default:
		e.Yol = []arayuz.Kirinti{{Metin: "DeltaX"}}
		sol, sag := m.girisGorunumu()
		govde = arayuz.Bolunmus(arayuz.Yigin(sol...), arayuz.Yigin(sag...))
		e.Durum = safe(m.identity.Name)
	}
	e.Diyalog = m.diyalogu()
	if bildirim := m.bildirimOgesi(); bildirim != nil {
		if e.Diyalog != nil && m.isErr {
			// Hata, kullanıcının baktığı diyaloğun içinde görünür.
			d := arayuz.Yigin(*bildirim)
			if e.Diyalog.Govde != nil {
				d.Cocuklar = append(d.Cocuklar, *e.Diyalog.Govde)
			}
			e.Diyalog.Govde = &d
		} else {
			govde = basinaEkle(govde, *bildirim)
		}
	}
	e.Govde = &govde
	return e
}

// basinaEkle, ileti kartını görünümün başına koyar: iki bölmeli görünümde
// sağ bölmenin başına.
func basinaEkle(govde, b arayuz.Oge) arayuz.Oge {
	switch {
	case govde.Tur == arayuz.TurBolunmus && govde.Sag != nil && govde.Sag.Tur == arayuz.TurYigin:
		sag := *govde.Sag
		sag.Cocuklar = append([]arayuz.Oge{b}, sag.Cocuklar...)
		govde.Sag = &sag
		return govde
	case govde.Tur == arayuz.TurYigin:
		govde.Cocuklar = append([]arayuz.Oge{b}, govde.Cocuklar...)
		return govde
	}
	return arayuz.Yigin(b, govde)
}

// bildirimOgesi, modelin son iletisidir (m.notice): hata ya da bilgi kartı.
func (m model) bildirimOgesi() *arayuz.Oge {
	if strings.TrimSpace(m.notice) == "" {
		return nil
	}
	stil := "iyi"
	if m.isErr {
		stil = "hata"
	}
	return new(arayuz.Kart(arayuz.Metin(strings.TrimSpace(m.notice), arayuz.IleStil(stil))))
}

// bolum, başlıklı ve isteğe bağlı araç düğmeli bir bölümdür.
func bolum(baslik string, araclar []arayuz.Oge, cocuklar ...arayuz.Oge) arayuz.Oge {
	b := arayuz.Bolum(baslik, cocuklar...)
	b.Araclar = araclar
	return b
}

// ilkSatirSafe, kayıt mesajının ilk satırıdır (temizlenmiş).
func ilkSatirSafe(s string) string { return safe(ilkSatir(s)) }

// ── Giriş ────────────────────────────────────────────────────────────────

func (m model) girisGorunumu() (sol, sag []arayuz.Oge) {
	var gruplar []arayuz.ListeOgesi
	for _, g := range m.gruplar {
		o := arayuz.ListeOgesi{ID: g.ID, Baslik: safe(g.Ad), Ikon: "ag",
			Alt: dil.F("%d üye", len(g.Uyeler)) + " · " + safe(g.KurucuAd)}
		switch {
		case g.benKurucuMuyum(m.identity.PeerID):
			o.Rozetler = []arayuz.Rozet{{Metin: dil.T("kurucu"), Renk: "vurgu"}}
		case g.yoneticiMi(m.identity.PeerID):
			o.Rozetler = []arayuz.Rozet{{Metin: dil.T("yönetici"), Renk: "vurgu"}}
		}
		gruplar = append(gruplar, o)
	}
	yeni := arayuz.Dugme("yeniGrup", dil.T("Yeni grup"), arayuz.Hayalet, arayuz.IleIkon("arti"))
	sol = append(sol, bolum(dil.T("Gruplar"), []arayuz.Oge{yeni},
		arayuz.Liste("gruplar", gruplar, arayuz.Aranabilir, arayuz.IleBos(dil.T("Henüz bir grubun yok.")))))
	if len(m.bekleyenDavetler) > 0 {
		var davetler []arayuz.ListeOgesi
		for _, d := range m.bekleyenDavetler {
			davetler = append(davetler, arayuz.ListeOgesi{ID: d.GrupID, Baslik: safe(d.Payload.GrupAd), Ikon: "gonder",
				Alt:      dil.F("Davet eden: %s", safe(d.Payload.Gonderen.Ad)),
				Dugmeler: []arayuz.Oge{arayuz.Dugme("davetKabul", dil.T("Kabul et"), arayuz.Birincil), arayuz.Dugme("davetRed", dil.T("Reddet"))}})
		}
		sol = append(sol, bolum(dil.T("Davetler"), nil, arayuz.Liste("davetler", davetler)))
	}

	if len(m.gruplar) == 0 {
		sag = append(sag, arayuz.Bos("kod", dil.T("Henüz bir grubun yok."),
			dil.T("Bir grup kur ve üyelerini davet et; projeleriniz yalnızca grup üyeleri arasında, uçtan uca şifreli eşitlenir."),
			arayuz.IleDugme(arayuz.Dugme("yeniGrup", dil.T("Yeni grup"), arayuz.Birincil, arayuz.IleIkon("arti")))))
	}
	sag = append(sag, arayuz.Kart(
		arayuz.Baslik(2, safe(m.identity.Name)),
		arayuz.AnahtarDeger(arayuz.Cift{Anahtar: "Peer ID", Deger: m.identity.PeerID, Kod: true, Kopyala: true}),
		arayuz.Metin(dil.T("Seni bir gruba davet etmek isteyen kişi bu Peer ID'yi kullanır."), arayuz.Soluk),
	))
	if len(m.sonKayitlar) > 0 {
		var kayitlar []arayuz.ListeOgesi
		for _, k := range m.sonKayitlar {
			kayitlar = append(kayitlar, arayuz.ListeOgesi{ID: k.grupID + "/" + k.projeID + "/" + k.Hash, Ikon: "kod",
				Baslik: ilkSatirSafe(k.Commit.Mesaj), Alt: safe(k.grup) + " / " + safe(k.proje) + " · " + safe(k.Commit.Yazar),
				Sag: k.Commit.TS.Format(zamanBicimi)})
		}
		sag = append(sag, bolum(dil.T("Son kayıtlar"), nil, arayuz.Liste("sonKayitlar", kayitlar)))
	}
	return sol, sag
}

// ── Grup ─────────────────────────────────────────────────────────────────

func (m model) grupGorunumu() (sol, sag []arayuz.Oge) {
	g := m.aktifGrup
	var projeler []arayuz.ListeOgesi
	for _, p := range m.projeler {
		o := arayuz.ListeOgesi{ID: p.ID, Baslik: safe(p.Ad), Ikon: "kod", Alt: dil.T("Henüz kayıt yok.")}
		if k, ok := m.projeSonKayit[p.ID]; ok {
			o.Alt = ilkSatirSafe(k.Commit.Mesaj)
			o.Sag = k.Commit.TS.Format(zamanBicimi)
		}
		projeler = append(projeler, o)
	}
	yeni := arayuz.Dugme("yeniProje", dil.T("Yeni proje"), arayuz.Hayalet, arayuz.IleIkon("arti"))
	sol = append(sol, bolum(dil.T("Projeler"), []arayuz.Oge{yeni},
		arayuz.Liste("projeler", projeler, arayuz.Aranabilir, arayuz.IleBos(dil.T("Bu grupta henüz proje yok.")))))

	sag = append(sag, arayuz.Kart(
		arayuz.Baslik(2, safe(g.Ad)),
		arayuz.AnahtarDeger(
			arayuz.Cift{Anahtar: dil.T("Kurucu"), Deger: safe(g.KurucuAd)},
			arayuz.Cift{Anahtar: dil.T("üye"), Deger: strconv.Itoa(len(g.Uyeler))},
		),
	))
	var uyeler []arayuz.ListeOgesi
	for _, u := range g.Uyeler {
		o := arayuz.ListeOgesi{ID: u.PeerID, Baslik: m.uyeAdi(u.PeerID), Alt: kisaPeer(u.PeerID), Ikon: "profil"}
		if r := rol(*g, u.PeerID); r != "" {
			o.Rozetler = append(o.Rozetler, arayuz.Rozet{Metin: r, Renk: "vurgu"})
		}
		switch {
		case u.PeerID == m.identity.PeerID:
			o.Rozetler = append(o.Rozetler, arayuz.Rozet{Metin: dil.T("sen")})
		case m.cevrimici[u.PeerID]:
			o.Sag = dil.T("çevrimiçi")
		default:
			o.Soluk = true
		}
		uyeler = append(uyeler, o)
	}
	var araclar []arayuz.Oge
	if g.yetkiliMi(m.identity.PeerID) {
		araclar = append(araclar, arayuz.Dugme("uyeDavet", dil.T("Üye davet et"), arayuz.Hayalet, arayuz.IleIkon("arti")))
	}
	if g.benKurucuMuyum(m.identity.PeerID) {
		araclar = append(araclar, arayuz.Dugme("yoneticiAta", dil.T("Yöneticiler"), arayuz.Hayalet, arayuz.IleIkon("kalkan")))
	}
	sag = append(sag, bolum(dil.T("Üyeler"), araclar, arayuz.Liste("uyeler", uyeler)))
	cik := dil.T("Gruptan çık")
	if g.yetkiliMi(m.identity.PeerID) {
		cik = dil.T("Grubu kapat")
	}
	sag = append(sag, arayuz.SatirOge(arayuz.Dugme("gruptanCik", cik, arayuz.Tehlikeli)))
	return sol, sag
}

// uyeDurumu, alt satırdaki çevrimiçi üye sayısıdır.
func (m model) uyeDurumu() string {
	if m.aktifGrup == nil {
		return ""
	}
	return strings.TrimSpace(dil.F(" %d/%d çevrimiçi", m.cevrimiciUyeSayisi(*m.aktifGrup), len(m.aktifGrup.Uyeler)))
}

// ── Diyaloglar ───────────────────────────────────────────────────────────

func vazgecDugmesi() arayuz.Oge { return arayuz.Dugme("vazgec", dil.T("Vazgeç")) }

func formDiyalogu(id, baslik, aciklama string, form arayuz.Oge, tamam string) *arayuz.Diyalog {
	govde := arayuz.Yigin(form)
	if aciklama != "" {
		govde.Cocuklar = append(govde.Cocuklar, arayuz.Metin(aciklama, arayuz.Soluk))
	}
	return &arayuz.Diyalog{ID: id, Baslik: baslik, Govde: &govde,
		Dugmeler: []arayuz.Oge{vazgecDugmesi(), arayuz.Dugme("tamam", tamam, arayuz.Birincil, arayuz.Gonderir)}}
}

// diyalogu, açık ekran bir diyalogsa onun tarifidir.
func (m model) diyalogu() *arayuz.Diyalog {
	switch m.ekran {
	case ekGrupOlustur:
		return formDiyalogu("yeniGrupD", dil.T("Yeni grup"),
			dil.T("Grup adını yaz. Grup için rastgele bir AES-256 anahtarı üretilir; yalnızca davet ettiğin üyeler bilir."),
			arayuz.Form("grupFormu", arayuz.Girdi("ad", dil.T("Ad"), m.grupAdIn.Value(), arayuz.Odakli, arayuz.Zorunlu, arayuz.IleEnUzun(80))),
			dil.T("Oluştur"))
	case ekProjeOlustur:
		return formDiyalogu("yeniProjeD", dil.T("Yeni proje"),
			dil.T("Proje adını yaz. Proje bu grubun bütün üyeleriyle paylaşılabilir."),
			arayuz.Form("projeFormu", arayuz.Girdi("ad", dil.T("Ad"), m.projeAdIn.Value(), arayuz.Odakli, arayuz.Zorunlu, arayuz.IleEnUzun(80))),
			dil.T("Oluştur"))
	case ekUyeDavet:
		var oneriler []arayuz.Oneri
		for _, p := range m.davetOnerileri() {
			oneriler = append(oneriler, arayuz.Oneri{Deger: p.PeerID, Etiket: kisaPeer(p.PeerID)})
		}
		return formDiyalogu("uyeDavetD", dil.T("Üye davet et"), "",
			arayuz.Form("davetFormu", arayuz.Girdi("peer", "Peer ID", m.uyeDavetIn.Value(), arayuz.Odakli, arayuz.Zorunlu,
				arayuz.IleYerTutucu("12D3KooW…"), arayuz.IleOneriler(oneriler...),
				arayuz.IleAciklama(dil.T("Kişi kendi Peer ID'sini DeltaX'in giriş ekranından paylaşabilir.")))),
			dil.T("Davet et"))
	case ekYoneticiSecim:
		var uyeler []arayuz.ListeOgesi
		for _, u := range m.yoneticiHar {
			o := arayuz.ListeOgesi{ID: u.PeerID, Baslik: m.uyeAdi(u.PeerID), Alt: kisaPeer(u.PeerID), Ikon: "profil"}
			if m.aktifGrup != nil && m.aktifGrup.yoneticiMi(u.PeerID) {
				o.Rozetler = []arayuz.Rozet{{Metin: dil.T("yönetici"), Renk: "vurgu"}}
			}
			uyeler = append(uyeler, o)
		}
		govde := arayuz.Yigin(
			arayuz.Metin(dil.T("Yöneticiler, kurucu gibi üye davet edebilir ve grubu kapatabilir: kurucuya ulaşılamazsa grup yönetimsiz kalmaz."), arayuz.Soluk),
			arayuz.Liste("yoneticiler", uyeler, arayuz.IleBos(dil.T("Grupta senden başka üye yok."))),
			arayuz.Metin(dil.T("Bir üyeye tıklayınca yöneticiliği açılır ya da kapanır."), arayuz.Soluk, arayuz.IleBoyut("kucuk")),
		)
		return &arayuz.Diyalog{ID: "yoneticilerD", Baslik: dil.T("Yöneticiler"), Govde: &govde,
			Dugmeler: []arayuz.Oge{arayuz.Dugme("vazgec", dil.T("Kapat"))}}
	case ekYeniDal:
		aciklama := ""
		if m.dalKaynak != "" {
			aciklama = dil.F("Yeni dal %s kaydından başlar.", kisaHash(m.dalKaynak))
		}
		return formDiyalogu("yeniDalD", dil.T("Yeni dal"), aciklama,
			arayuz.Form("dalFormu", arayuz.Girdi("ad", dil.T("Ad"), m.yeniDalIn.Value(), arayuz.Odakli, arayuz.Zorunlu, arayuz.IleEnUzun(60))),
			dil.T("Oluştur"))
	case ekOnay:
		baslik, satirlar := m.onayMetni()
		if baslik == "" {
			return nil
		}
		govde := arayuz.Yigin()
		for i, s := range satirlar {
			if i == 0 {
				govde.Cocuklar = append(govde.Cocuklar, arayuz.Metin(s, arayuz.Kalin))
			} else {
				govde.Cocuklar = append(govde.Cocuklar, arayuz.Metin(s, arayuz.Soluk))
			}
		}
		return &arayuz.Diyalog{ID: "onayD" + strconv.Itoa(int(m.onayTip)), Baslik: baslik, Govde: &govde,
			Dugmeler: []arayuz.Oge{arayuz.Dugme("onayHayir", dil.T("Vazgeç")),
				arayuz.Dugme("onayEvet", baslik, arayuz.Tehlikeli, arayuz.DevreDisiIse(m.pending))}}
	}
	return nil
}
