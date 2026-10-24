package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/lucian95511/and/dil/diltest"
)

// TestGorunum_HerDilde: DeltaX'in ekranları her dilde terminal genişliğini
// aşmaz ve çevrilmemiş metin içermez. Test verisindeki kullanıcı içeriği
// (üye, grup ve kayıt adları) özel addır.
func TestGorunum_HerDilde(t *testing.T) {
	denetci := diltest.YeniDenetci(t,
		[]string{"ayşe", "kurucu", "İlk sürüm", "Başlık güncellendi", "Merhaba dünya"},
		diltest.Oku(t, "ceviri.json", ceviriVerisi))
	diltest.HerDilde(t, func(kod string) {
		for _, boyut := range [][2]int{{80, 24}, {140, 40}} {
			m, _, _ := ornekProje(t, boyut[0], boyut[1])
			grup := bas(t, m, "esc")
			ana := bas(t, grup, "esc", "esc")
			ekranlar := map[string]model{
				"proje":         m,
				"degisiklikler": bas(t, m, "d"),
				"commit":        bas(t, m, "c"),
				"gunluk":        bas(t, m, "l"),
				"kayit":         bas(t, m, "l", "enter"),
				"kayit-fark":    bas(t, m, "l", "enter", "enter"),
				"dallar":        bas(t, m, "b"),
				"yeni-dal":      bas(t, m, "b", "n"),
				"uzak":          bas(t, m, "m"),
				"gonder":        bas(t, m, "p"),
				"pull":          bas(t, m, "u"),
				"dosyalar":      bas(t, m, "f"),
				"disa-aktar":    bas(t, m, "e"),
				"grup":          grup,
				"davet":         bas(t, grup, "i"),
				"yoneticiler":   bas(t, grup, "y"),
				"grubu-kapat":   bas(t, grup, "x"),
				"ana":           ana,
				"yardim":        bas(t, ana, "?"),
				"gruplar":       bas(t, ana, "enter"),
				"yeni-grup":     bas(t, ana, "down", "enter"),
				"davetler":      bas(t, ana, "down", "down", "enter"),
				"kimlik":        bas(t, ana, "down", "down", "down", "enter"),
			}
			dokum := os.Getenv("DELTAX_EKRAN_DOKUM")
			for ad, e := range ekranlar {
				metin := ekranMetni(t, e)
				for _, k := range denetci.Kalintilar(kod, metin) {
					t.Errorf("%s %s %dx%d: %s", kod, ad, boyut[0], boyut[1], k)
				}
				if dokum != "" {
					_ = os.WriteFile(filepath.Join(dokum, kod+"_"+ad+"_"+strconv.Itoa(boyut[0])+".txt"), []byte(metin), 0o600)
				}
			}
		}
	})
}
