package main

// isleyiciler.go — Update, gelen her mesaj türünü kendi işleyicisine
// yönlendirir; işleyiciler aşağıda aynı sırayla durur. Bir türün
// işleyicisi yoksa mesaj açık ekrandaki giriş kutusuna iletilir.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/internal/ui"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.boyutDegisti(msg)
	case spinner.TickMsg:
		return m.spinnerTik(msg)
	case dmGecmisMsg:
		return m.dmGecmisiGeldi(msg)
	case dmPollResultMsg:
		return m.dmlerGeldi(msg)
	case filePollResultMsg:
		return m.dosyalarGeldi(msg)
	case consentPollResultMsg:
		return m.dosyaOnayiIstendi(msg)
	case peersResultMsg:
		return m.eslerGeldi(msg)
	case basitSonuc:
		return m.basitSonucGeldi(msg)
	case grupOlusturSonucu:
		return m.grupOlusturuldu(msg)
	case kabulSonucu:
		return m.davetKabulEdildi(msg)
	case projeOlusturSonucu:
		return m.projeOlusturuldu(msg)
	case durumSonucu:
		return m.durumGeldi(msg)
	case commitSonucu:
		return m.commitBitti(msg)
	case gunlukSonucu:
		return m.gunlukGeldi(msg)
	case dalDegistirSonucu:
		return m.dalDegisti(msg)
	case diffSonucu:
		return m.diffGeldi(msg)
	case pushSonucu:
		return m.pushBitti(msg)
	case birlestirSonucu:
		return m.birlestirmeBitti(msg)
	case grupCikKapatSonucu:
		return m.gruptanCikildi(msg)
	case dosyaListesiSonucu:
		return m.dosyaListesiGeldi(msg)
	case dosyaAcSonucu:
		return m.dosyaAcildi(msg)
	case dosyaKaydetSonucu:
		return m.dosyaKaydedildi(msg)
	case dosyaSilSonucu:
		return m.dosyaSilindi(msg)
	case disaAktarSonucu:
		return m.disaAktarildi(msg)
	case tea.KeyMsg:
		return m.tusaBasildi(msg)
	}
	return m.girdiyeIlet(msg)
}

func (m model) boyutDegisti(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.w, m.h = msg.Width, msg.Height
	// Ekranlar tam ekran çerçevede (bkz. tamEkran): kenarlık ve dolgu
	// düşülür; başlık, ayraç ve kısayol satırı için de yer bırakılır.
	cw, ch := ui.ContentWidth(msg.Width), max(msg.Height-12, 3)
	m.gunlukVP.Width = cw
	m.gunlukVP.Height = ch
	m.diffVP.Width = cw
	m.diffVP.Height = ch
	m.dosyaDuzenleTA.SetWidth(cw)
	m.dosyaDuzenleTA.SetHeight(ch)
	return m, nil
}

func (m model) spinnerTik(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if !m.pending {
		return m, nil // işlem bitti — tik zincirini burada durdur, boşuna dönmeye devam etmesin
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// dmlerGeldi, geçmişi olmayan eski bir AND'da canlı gelen DM'leri işler
// (bkz. dmGecmisiGeldi).
func (m model) dmlerGeldi(msg dmPollResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, dmPollCmdAfter(m.client)
	}
	var cmds []tea.Cmd
	for _, dm := range msg.msgs {
		if z, ok := zarfCoz(dm.Text); ok {
			var cmd tea.Cmd
			m, cmd = m.zarfIsle(z, dm.VerifiedPeerID, time.Now())
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	cmds = append(cmds, dmPollCmd(m.client))
	return m, tea.Batch(cmds...)
}

func (m model) dosyalarGeldi(msg filePollResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, filePollCmdAfter(m.client)
	}
	for _, fm := range msg.msgs {
		if !strings.HasPrefix(filepath.Base(fm.Filename), "projeag-") {
			continue
		}
		_, projeID, manifest, sayisi, err := pushPaketiIsle(m.dataDir, fm.SavePath, fm.From)
		_ = os.Remove(fm.SavePath)
		if err != nil {
			m = m.bildirGuncelle("Paket işlenemedi: "+describeErr(err), true)
			continue
		}
		if p, perr := projeOku(m.dataDir, projeID); perr == nil {
			if p.Uzaklar == nil {
				p.Uzaklar = map[string]string{}
			}
			if manifest.HeadHash != "" {
				p.Uzaklar[fm.From+"/"+manifest.Dal] = manifest.HeadHash
			}
			_ = projeKaydet(m.dataDir, p)
			if m.aktifProje != nil && m.aktifProje.ID == projeID {
				*m.aktifProje = p
			}
		}
		m = m.bildirGuncelle(fmt.Sprintf("%d yeni nesne alındı (%s) — Uzak Dallar'dan birleştirebilirsin.", sayisi, safe(manifest.Dal)), false)
	}
	return m, filePollCmd(m.client)
}

func (m model) dosyaOnayiIstendi(msg consentPollResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil || len(msg.reqs) == 0 {
		return m, consentPollCmd(m.client)
	}
	req := msg.reqs[0]
	if strings.HasPrefix(req.Filename, "projeag-") && paketGonderenGuvenilirMi(m.dataDir, req.Filename, req.SenderID) {
		return m, tea.Batch(onayCmd(m.client, req.TransferID, true), consentPollCmd(m.client))
	}
	m.pendingConsent = &req
	m.onayOnceki = m.ekran
	m.ekran = ekDosyaOnay
	return m, nil
}

func (m model) eslerGeldi(msg peersResultMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil {
		nc := make(map[string]bool, len(msg.peers))
		for _, p := range msg.peers {
			nc[p.PeerID] = true
		}
		m.cevrimici = nc
	}
	return m, peersPollCmdAfter(m.client)
}

func (m model) basitSonucGeldi(msg basitSonuc) (tea.Model, tea.Cmd) {
	m.pending = false
	switch msg.islem {
	case "davet":
		if msg.err != nil {
			m.notice, m.isErr = "Davet gönderilemedi: "+describeErr(msg.err), true
		} else {
			m.notice, m.isErr = "Davet gönderildi. Kişi çevrimdışıysa AND daveti bekletir, bağlandığında iletir.", false
		}
	case "pull_iste":
		if msg.err != nil {
			m.notice, m.isErr = "Çek isteği gönderilemedi: "+describeErr(msg.err), true
		} else {
			m.notice, m.isErr = "Çek isteği gönderildi — karşı taraf çevrimiçiyse kısa süre içinde değişiklikler gelecek.", false
		}
	case "roster":
		if msg.err != nil {
			m.notice, m.isErr = "Üye listesi güncellemesi bazı üyelere ulaşmadı.", true
		}
	}
	return m, nil
}

func (m model) grupOlusturuldu(msg grupOlusturSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "Grup oluşturulamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.notice, m.isErr = "\""+msg.grup.Ad+"\" grubu oluşturuldu.", false
	m.gruplar = gruplariListele(m.dataDir)
	m.ekran = ekGrupListesi
	return m, nil
}

func (m model) davetKabulEdildi(msg kabulSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "İşlem hatası: "+describeErr(msg.err), true
		return m, nil
	}
	if msg.grup.ID != "" {
		m.notice, m.isErr = "\""+msg.grup.Ad+"\" grubuna katıldın.", false
		m.gruplar = gruplariListele(m.dataDir)
	} else {
		m.notice, m.isErr = "Davet reddedildi.", false
	}
	return m, nil
}

func (m model) projeOlusturuldu(msg projeOlusturSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "Proje oluşturulamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.notice, m.isErr = "\""+msg.proje.Ad+"\" projesi oluşturuldu.", false
	m.projeler = projeleriListele(m.dataDir, m.aktifGrup.ID)
	m.ekran = ekGrupPaneli
	return m, nil
}

func (m model) durumGeldi(msg durumSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if !m.projeSonucuGuncelMi(msg.projeID) {
		return m, nil
	}
	if msg.err != nil {
		m.notice, m.isErr = "Durum hesaplanamadı: "+describeErr(msg.err), true
		m.ekran = ekProjePaneli
		return m, nil
	}
	m.durum = msg.durum
	if m.durumSonraki == ekDiffListesi {
		var dosyalar []string
		dosyalar = append(dosyalar, msg.durum.Degisen...)
		dosyalar = append(dosyalar, msg.durum.Eklenen...)
		sort.Strings(dosyalar)
		m.diffDosyalar = dosyalar
		m.diffIdx = 0
		m.ekran = ekDiffListesi
		return m, nil
	}
	m.ekran = ekDurum
	return m, nil
}

func (m model) commitBitti(msg commitSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if !m.projeSonucuGuncelMi(msg.projeID) {
		return m, nil
	}
	if msg.err != nil {
		m.notice, m.isErr = "Kayıt oluşturulamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.aktifProje.Dallar[m.aktifProje.AktifDal] = msg.hash
	_ = projeKaydet(m.dataDir, *m.aktifProje)
	if msg.atlanan > 0 {
		m.notice = fmt.Sprintf("Kayıt oluşturuldu: %s (%d büyük dosya atlandı)", kisaHash(msg.hash), msg.atlanan)
	} else {
		m.notice = "Kayıt oluşturuldu: " + kisaHash(msg.hash)
	}
	m.isErr = false
	m.commitMsgIn.SetValue("")
	m.ekran = ekProjePaneli
	return m, nil
}

func (m model) gunlukGeldi(msg gunlukSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	m.gunluk = msg.girdiler
	m.gunlukAramaAktif = false
	m.gunlukAramaIn.SetValue("")
	m.gunlukAramaIn.Blur()
	m.gunlukVP.SetContent(gunlukMetni(msg.girdiler))
	m.gunlukVP.GotoTop()
	m.ekran = ekGunluk
	return m, nil
}

func (m model) dalDegisti(msg dalDegistirSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if !m.projeSonucuGuncelMi(msg.projeID) {
		return m, nil
	}
	if msg.err != nil {
		m.notice, m.isErr = "Dal değiştirilemedi: "+describeErr(msg.err), true
		return m, nil
	}
	if msg.kirli {
		m.notice, m.isErr = "Önce mevcut değişiklikleri kaydet (kaydedilmemiş dosyalar var).", true
		return m, nil
	}
	m.aktifProje.AktifDal = msg.dal
	_ = projeKaydet(m.dataDir, *m.aktifProje)
	m.notice, m.isErr = "Dal değiştirildi: "+msg.dal, false
	m.ekran = ekProjePaneli
	return m, nil
}

func (m model) diffGeldi(msg diffSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = describeErr(msg.err), true
		m.ekran = ekDiffListesi
		return m, nil
	}
	m.diffVP.SetContent(diffMetni(msg.satirlar))
	m.diffVP.GotoTop()
	m.ekran = ekDiff
	return m, nil
}

func (m model) pushBitti(msg pushSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = safe(msg.hedefAd)+": gönderilemedi — "+describeErr(msg.err), true
	} else {
		m.notice, m.isErr = fmt.Sprintf("%s kullanıcısına %d nesne gönderildi.", safe(msg.hedefAd), msg.nesneSayisi), false
		if m.aktifProje != nil {
			p, err := projeOku(m.dataDir, m.aktifProje.ID)
			if err == nil {
				*m.aktifProje = p
			}
		}
	}
	return m, nil
}

func (m model) birlestirmeBitti(msg birlestirSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if !m.projeSonucuGuncelMi(msg.projeID) {
		return m, nil
	}
	if msg.err != nil {
		m.notice, m.isErr = "Birleştirme hatası: "+describeErr(msg.err), true
		return m, nil
	}
	if len(msg.catismalar) > 0 {
		m.sonCatismalar = msg.catismalar
		m.notice = fmt.Sprintf("%d dosyada çakışma var — dosyaları elle düzenleyip Kaydet'e bas.", len(msg.catismalar))
		m.isErr = true
		m.ekran = ekProjePaneli
		return m, nil
	}
	m.aktifProje.Dallar[m.aktifProje.AktifDal] = msg.yeniCommit
	_ = projeKaydet(m.dataDir, *m.aktifProje)
	m.notice, m.isErr = "Birleştirme tamamlandı: "+kisaHash(msg.yeniCommit), false
	m.ekran = ekProjePaneli
	return m, nil
}

func (m model) gruptanCikildi(msg grupCikKapatSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "İşlem başarısız: "+describeErr(msg.err), true
		return m, nil
	}
	m.aktifGrup = nil
	m.aktifProje = nil
	m.gruplar = gruplariListele(m.dataDir)
	if m.grupIdx >= len(m.gruplar) {
		m.grupIdx = max0(len(m.gruplar) - 1)
	}
	m.ekran = ekGrupListesi
	if msg.kapatildi {
		m.notice = "Grup kapatıldı."
	} else {
		m.notice = "Gruptan çıkıldı."
	}
	if msg.agHatasi != nil {
		m.notice += " (uyarı: " + describeErr(msg.agHatasi) + ")"
	}
	m.isErr = false
	return m, nil
}

func (m model) dosyaListesiGeldi(msg dosyaListesiSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "Dosya listesi alınamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.dosyaListesi = msg.liste
	if m.dosyaIdx >= len(m.dosyaListesi) {
		m.dosyaIdx = max0(len(m.dosyaListesi) - 1)
	}
	m.dosyaSilOnayi = ""
	m.ekran = ekDosyaListesi
	return m, nil
}

func (m model) dosyaAcildi(msg dosyaAcSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "Dosya açılamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.dosyaDuzenleYol = msg.rel
	m.dosyaDuzenleTA.SetValue(msg.icerik)
	m.dosyaDuzenleTA.Focus()
	m.ekran = ekDosyaDuzenle
	return m, nil
}

func (m model) dosyaKaydedildi(msg dosyaKaydetSonucu) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.notice, m.isErr = "Kaydedilemedi: "+describeErr(msg.err), true
		return m, nil
	}
	m.notice, m.isErr = "Kaydedildi: "+m.dosyaDuzenleYol, false
	return m, nil
}

func (m model) dosyaSilindi(msg dosyaSilSonucu) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.notice, m.isErr = "Silinemedi: "+describeErr(msg.err), true
		return m, nil
	}
	m.notice, m.isErr = "Silindi: "+msg.rel, false
	return m, dosyaListesiCmd(m.calismaDir())
}

func (m model) disaAktarildi(msg disaAktarSonucu) (tea.Model, tea.Cmd) {
	m.pending = false
	if msg.err != nil {
		m.notice, m.isErr = "Dışa aktarılamadı: "+describeErr(msg.err), true
		return m, nil
	}
	m.notice, m.isErr = "Dışa aktarıldı: "+msg.yol, false
	m.ekran = ekProjePaneli
	return m, nil
}

func (m model) tusaBasildi(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ekran == ekYardim {
		m.ekran = m.yardimGeriEkran
		return m, nil
	}
	if msg.String() == "?" && len(m.kisayollarIcin(m.ekran)) > 0 {
		m.yardimGeriEkran = m.ekran
		m.ekran = ekYardim
		return m, nil
	}
	return m.handleKey(msg)
}

// girdiyeIlet, mesajı açık ekrandaki giriş kutusuna ya da görünüme iletir
// (imleç yanıp sönmesi, yazılan karakterler, kaydırma).
func (m model) girdiyeIlet(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.ekran {
	case ekGrupOlustur:
		m.grupAdIn, cmd = m.grupAdIn.Update(msg)
	case ekGrupListesi:
		if m.grupAramaAktif {
			m.grupAramaIn, cmd = m.grupAramaIn.Update(msg)
		}
	case ekUyeDavet:
		m.uyeDavetIn, cmd = m.uyeDavetIn.Update(msg)
	case ekProjeOlustur:
		m.projeAdIn, cmd = m.projeAdIn.Update(msg)
	case ekCommitMesaj:
		m.commitMsgIn, cmd = m.commitMsgIn.Update(msg)
	case ekYeniDal:
		m.yeniDalIn, cmd = m.yeniDalIn.Update(msg)
	case ekGunluk:
		if m.gunlukAramaAktif {
			m.gunlukAramaIn, cmd = m.gunlukAramaIn.Update(msg)
		} else {
			m.gunlukVP, cmd = m.gunlukVP.Update(msg)
		}
	case ekDiff:
		m.diffVP, cmd = m.diffVP.Update(msg)
	case ekYeniDosya:
		m.yeniDosyaIn, cmd = m.yeniDosyaIn.Update(msg)
	case ekDosyaDuzenle:
		m.dosyaDuzenleTA, cmd = m.dosyaDuzenleTA.Update(msg)
	case ekDisaAktar:
		m.disaAktarIn, cmd = m.disaAktarIn.Update(msg)
	}
	return m, cmd
}
