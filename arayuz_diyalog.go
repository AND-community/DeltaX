package main

// arayuz_diyalog.go — arayüz kipinde Uzak ve Dosyalar sekmeleri ile proje
// diyalogları (gönder, çek, dışa aktar, yeni dosya) ve gelen dosya onayı.

import (
	"strings"

	"github.com/lucian95511/and/arayuz"
	"github.com/lucian95511/and/dil"
)

func (m model) uzakSekmesi() arayuz.Oge {
	var ogeler []arayuz.ListeOgesi
	for _, anahtar := range m.uzakAnahtarlari {
		peer, dal, _ := strings.Cut(anahtar, "/")
		o := arayuz.ListeOgesi{ID: anahtar, Baslik: m.uyeAdi(peer) + "  ⎇ " + safe(dal), Ikon: "kure",
			Alt:      kisa7(m.aktifProje.Uzaklar[anahtar]),
			Dugmeler: []arayuz.Oge{arayuz.Dugme("birlestir", dil.T("Birleştir"))}}
		if m.ozet != nil {
			if n := m.ozet.uzakYeni[anahtar]; n > 0 {
				o.Rozetler = []arayuz.Rozet{{Metin: dil.F("%d yeni", n), Renk: "iyi"}}
			}
		}
		ogeler = append(ogeler, o)
	}
	return bolum(dil.T("Uzak dallar"),
		[]arayuz.Oge{arayuz.Dugme("herkestenCek", dil.T("Çevrimiçi herkesten çek"), arayuz.Hayalet, arayuz.IleIkon("asagi"), arayuz.DevreDisiIse(m.pending))},
		arayuz.Liste("uzaklar", ogeler, arayuz.IleBos(dil.T("Henüz uzak dal yok. Bir üye değişikliklerini gönderince burada görünür."))))
}

func (m model) dosyalarSekmesi(arka ekran) arayuz.Oge {
	acik := ""
	if arka == ekDosyaDuzenle {
		acik = m.dosyaDuzenleYol
	}
	var ogeler []arayuz.ListeOgesi
	for _, yol := range m.dosyaListesi {
		ogeler = append(ogeler, arayuz.ListeOgesi{ID: yol, Baslik: safe(yol), Ikon: "belge",
			Dugmeler: []arayuz.Oge{arayuz.Dugme("dosyaSil", dil.T("Sil"), arayuz.Tehlikeli,
				arayuz.IleOnay(dil.T("Dosya silinsin mi?"), safe(yol), dil.T("Sil")))}})
	}
	sol := bolum(dil.T("Dosyalar"), []arayuz.Oge{arayuz.Dugme("yeniDosya", dil.T("Yeni dosya"), arayuz.Hayalet, arayuz.IleIkon("arti"))},
		arayuz.Liste("dosyalar", ogeler, arayuz.IleSecili(acik), arayuz.Aranabilir, arayuz.IleBos(dil.T("Çalışma klasörü boş."))))
	if acik == "" {
		return arayuz.Bolunmus(sol, arayuz.Bos("belge", dil.T("Bir dosya seç"), dil.T("Seçtiğin dosya burada düzenlenir.")))
	}
	return arayuz.Bolunmus(sol, arayuz.Form("duzenleyici",
		arayuz.MetinAlani("icerik", safe(acik), m.duzenleyiciIcerik, arayuz.KodGibi, arayuz.IleSatirSayisi(24)),
		arayuz.SatirOge(
			arayuz.Dugme("dosyaKaydet", dil.T("Kaydet"), arayuz.Birincil, arayuz.Gonderir, arayuz.IleIkon("tamam"), arayuz.IleKisayol("ctrl+s")),
			arayuz.Dugme("duzenleyiciKapat", dil.T("Kapat"), arayuz.Hayalet),
		)))
}

// uyeSecimListesi, gönder/çek diyaloğundaki üye listesidir.
func (m model) uyeSecimListesi(id string) arayuz.Oge {
	var ogeler []arayuz.ListeOgesi
	for _, u := range m.uyelerHar {
		o := arayuz.ListeOgesi{ID: u.PeerID, Baslik: m.uyeAdi(u.PeerID), Alt: kisaPeer(u.PeerID), Ikon: "profil"}
		if m.cevrimici[u.PeerID] {
			o.Sag = dil.T("çevrimiçi")
		} else {
			o.Soluk = true
		}
		ogeler = append(ogeler, o)
	}
	return arayuz.Liste(id, ogeler, arayuz.IleBos(dil.T("Grupta senden başka üye yok.")))
}

// projeDiyalogu, proje görünümüne ait diyalogdur (yoksa nil).
func (m model) projeDiyalogu() *arayuz.Diyalog {
	switch m.ekran {
	case ekGonderSecim:
		govde := arayuz.Yigin(
			arayuz.Metin(dil.T("Yeni kayıtlarını seçtiğin üyeye grup anahtarıyla şifreli bir paket olarak gönderir. Üyenin çevrimiçi olması gerekir."), arayuz.Soluk),
			m.uyeSecimListesi("gonderHedefleri"))
		return &arayuz.Diyalog{ID: "gonderD", Baslik: dil.T("Gönder"), Govde: &govde, Dugmeler: []arayuz.Oge{vazgecDugmesi(),
			arayuz.Dugme("herkeseGonder", dil.T("Çevrimiçi herkese gönder"), arayuz.Birincil, arayuz.DevreDisiIse(m.pending))}}
	case ekPullIste:
		govde := arayuz.Yigin(
			arayuz.Metin(dil.T("Seçtiğin üyeden kendi değişikliklerini sana göndermesini ister. Üyenin çevrimiçi olması gerekir."), arayuz.Soluk),
			m.uyeSecimListesi("cekHedefleri"))
		return &arayuz.Diyalog{ID: "cekD", Baslik: dil.T("Çek"), Govde: &govde, Dugmeler: []arayuz.Oge{vazgecDugmesi(),
			arayuz.Dugme("herkestenCekDiyalog", dil.T("Çevrimiçi herkesten çek"), arayuz.Birincil, arayuz.DevreDisiIse(m.pending))}}
	case ekDisaAktar:
		return formDiyalogu("disaAktarD", dil.T("Dışa aktar"), dil.T("Proje, seçtiğin klasöre bir .zip arşivi olarak yazılır."),
			arayuz.Form("disaAktarFormu", arayuz.Girdi("hedef", dil.T("Hedef klasör"), m.disaAktarIn.Value(), arayuz.Odakli, arayuz.Zorunlu, arayuz.IleEnUzun(300))),
			dil.T("Dışa aktar"))
	case ekYeniDosya:
		return formDiyalogu("yeniDosyaD", dil.T("Yeni dosya"), "",
			arayuz.Form("yeniDosyaFormu", arayuz.Girdi("yol", dil.T("Dosya yolu"), m.yeniDosyaIn.Value(), arayuz.Odakli, arayuz.Zorunlu,
				arayuz.IleYerTutucu("klasor/dosya.txt"))),
			dil.T("Oluştur"))
	case ekDosyaOnay:
		c := m.pendingConsent
		if c == nil {
			return nil
		}
		govde := arayuz.Yigin(
			arayuz.AnahtarDeger(
				arayuz.Cift{Anahtar: dil.T("Gönderen"), Deger: kisaPeer(safe(c.SenderID)), Kod: true},
				arayuz.Cift{Anahtar: dil.T("Dosya"), Deger: safe(c.Filename)},
				arayuz.Cift{Anahtar: dil.T("Boyut"), Deger: boyutYaz(c.Size)},
			),
			arayuz.Metin(dil.T("Bu gönderen, bulunduğun hiçbir grubun üyesi değil."), arayuz.Soluk),
		)
		return &arayuz.Diyalog{ID: "dosyaOnayD" + c.TransferID, Baslik: dil.T("Bilinmeyen gönderenden dosya"), Govde: &govde,
			Dugmeler: []arayuz.Oge{arayuz.Dugme("dosyaOnayHayir", dil.T("Reddet")), arayuz.Dugme("dosyaOnayEvet", dil.T("Kabul et"), arayuz.Birincil)}}
	}
	return nil
}
