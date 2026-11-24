package main

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/arayuz"
	"github.com/lucian95511/and/arayuz/arayuztest"
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
