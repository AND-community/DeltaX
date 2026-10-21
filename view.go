package main

// view.go — ekranların görünümü. Görsel dil AND'ın kendisiyle ortaktır
// (internal/ui): tam ekran çerçeve, konum izli başlık çubuğu, seçim
// çubuklu liste satırları, satır içi bildirimler ve altta kısayol satırı.
// Her ekran gövdesini ve kısayollarını döndürür; çerçeveyi View kurar.
// Onay, dosya isteği ve yardım kutuları ekranın ortasında açılır.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lucian95511/and/internal/ui"
)

// icW, tam ekran çerçevenin içerik genişliğidir.
func (m model) icW() int {
	if m.w <= 0 {
		return 72
	}
	return ui.ContentWidth(m.w)
}

// govdeYuksekligi, başlık çubuğu ve alt bilgi dışında gövdeye kalan satır sayısıdır.
func (m model) govdeYuksekligi() int {
	if m.h <= 0 {
		return 20
	}
	return max(ui.ContentHeight(m.h)-7, 4)
}

func (m model) View() string {
	switch m.ekran {
	case ekDosyaOnay:
		return m.viewDosyaOnay()
	case ekOnay:
		return m.viewOnay()
	case ekYardim:
		return m.viewYardim()
	}
	w := m.icW()
	govde, ipuclari := m.govde(w)
	if len(m.kisayollarIcin(m.ekran)) > 0 {
		ipuclari = append(ipuclari, ui.H("?", "yardım"))
	}

	var b strings.Builder
	b.WriteString(ui.Header(w, m.konumIzi(), m.baslikSag()) + "\n\n")
	b.WriteString(govde)

	var f strings.Builder
	f.WriteString(ui.Divider(w) + "\n")
	if s := m.durumSatiri(); s != "" {
		f.WriteString(ui.Truncate(s, w) + "\n")
	}
	f.WriteString(ui.KeyHints(w, ipuclari...))
	return ui.Page(m.w, m.h, b.String(), f.String())
}

func (m model) govde(w int) (string, []ui.Hint) {
	switch m.ekran {
	case ekAnaMenu:
		return m.viewAnaMenu(w)
	case ekKimlik:
		return m.viewKimlik(w)
	case ekGrupListesi:
		return m.viewGrupListesi(w)
	case ekGrupOlustur:
		return m.viewGirdi("Grup adını yaz. Grup için rastgele bir AES-256 anahtarı üretilir; yalnızca davet ettiğin üyeler bilir.",
			m.grupAdIn.View(), ui.H("enter", "oluştur"), ui.H("esc", "vazgeç"))
	case ekGrupPaneli:
		return m.viewGrupPaneli(w)
	case ekUyeDavet:
		return m.viewUyeDavet(w)
	case ekDavetler:
		return m.viewDavetler(w)
	case ekProjeOlustur:
		return m.viewGirdi("Proje adını yaz. Proje bu grubun bütün üyeleriyle paylaşılabilir.",
			m.projeAdIn.View(), ui.H("enter", "oluştur"), ui.H("esc", "vazgeç"))
	case ekProjePaneli:
		return m.viewProjePaneli(w)
	case ekDurum:
		return m.viewDurum(w)
	case ekCommitMesaj:
		return m.viewCommitMesaj()
	case ekGunluk:
		return m.viewGunluk(w)
	case ekKayit:
		return m.viewKayit(w)
	case ekDallar:
		return m.viewDallar(w)
	case ekYeniDal:
		return m.viewYeniDal()
	case ekDiff:
		return m.viewDiff()
	case ekGonderSecim:
		return m.viewUyeSecimi(w, m.pushIdx, "Yeni kayıtlarını seçtiğin üyeye grup anahtarıyla şifreli bir paket olarak gönderir. Üyenin çevrimiçi olması gerekir.",
			ui.H("↑↓", "seç"), ui.H("enter", "gönder"), ui.H("a", "çevrimiçi herkese"), ui.H("esc", "geri"))
	case ekUzakDallar:
		return m.viewUzakDallar(w)
	case ekPullIste:
		return m.viewUyeSecimi(w, m.pullIdx, "Seçtiğin üyeden kendi değişikliklerini sana göndermesini ister. Üyenin çevrimiçi olması gerekir.",
			ui.H("↑↓", "seç"), ui.H("enter", "iste"), ui.H("a", "çevrimiçi herkesten"), ui.H("esc", "geri"))
	case ekYoneticiSecim:
		return m.viewYoneticiSecim(w)
	case ekDosyaListesi:
		return m.viewDosyaListesi(w)
	case ekYeniDosya:
		return m.viewGirdi("Çalışma klasörüne göre dosya yolu (ör. src/ana.go). Klasörler gerekirse oluşturulur.",
			m.yeniDosyaIn.View(), ui.H("enter", "oluştur ve düzenle"), ui.H("esc", "vazgeç"))
	case ekDosyaDuzenle:
		return m.dosyaDuzenleTA.View() + "\n", []ui.Hint{ui.H("ctrl+s", "kaydet"), ui.H("esc", "listeye dön")}
	case ekDisaAktar:
		return m.viewGirdi("Projenin tüm geçmişi (nesneler ve çalışma klasörü) bu klasöre tek bir .zip dosyası olarak yazılır.",
			m.disaAktarIn.View(), ui.H("enter", "dışa aktar"), ui.H("esc", "vazgeç"))
	}
	return "", nil
}

// ─── Başlık ve alt bilgi ────────────────────────────────────────────────────

var ekranAdlari = map[ekran][]string{
	ekKimlik:        {"Kimlik"},
	ekGrupListesi:   {"Gruplarım"},
	ekGrupOlustur:   {"Yeni grup"},
	ekDavetler:      {"Davetler"},
	ekUyeDavet:      {"Üye davet et"},
	ekProjeOlustur:  {"Yeni proje"},
	ekYoneticiSecim: {"Yöneticiler"},
	ekDurum:         {"Değişiklikler"},
	ekCommitMesaj:   {"Kaydet"},
	ekGunluk:        {"Günlük"},
	ekDallar:        {"Dallar"},
	ekYeniDal:       {"Dallar", "Yeni dal"},
	ekGonderSecim:   {"Gönder"},
	ekUzakDallar:    {"Uzak dallar"},
	ekPullIste:      {"Çek iste"},
	ekDosyaListesi:  {"Dosyalar"},
	ekYeniDosya:     {"Dosyalar", "Yeni dosya"},
	ekDisaAktar:     {"Dışa aktar"},
}

// projeEkraniMi, ekranın bir projenin içinde olup olmadığıdır.
func projeEkraniMi(e ekran) bool {
	switch e {
	case ekProjePaneli, ekDurum, ekCommitMesaj, ekGunluk, ekKayit, ekDallar, ekYeniDal, ekDiff,
		ekGonderSecim, ekUzakDallar, ekPullIste, ekDosyaListesi, ekYeniDosya, ekDosyaDuzenle, ekDisaAktar:
		return true
	}
	return false
}

func grupEkraniMi(e ekran) bool {
	switch e {
	case ekGrupPaneli, ekUyeDavet, ekProjeOlustur, ekYoneticiSecim:
		return true
	}
	return false
}

func (m model) konumIzi() []string {
	iz := []string{"DeltaX"}
	if (grupEkraniMi(m.ekran) || projeEkraniMi(m.ekran)) && m.aktifGrup != nil {
		iz = append(iz, safe(m.aktifGrup.Ad))
	}
	if projeEkraniMi(m.ekran) && m.aktifProje != nil {
		iz = append(iz, safe(m.aktifProje.Ad))
	}
	switch m.ekran {
	case ekKayit:
		return append(iz, "Günlük", kisa7(m.kayit.hash))
	case ekDiff:
		if m.diffGeri == ekKayit {
			iz = append(iz, "Günlük", kisa7(m.kayit.hash))
		} else {
			iz = append(iz, "Değişiklikler")
		}
		return append(iz, safe(m.diffBaslik))
	case ekDosyaDuzenle:
		return append(iz, "Dosyalar", safe(m.dosyaDuzenleYol))
	}
	return append(iz, ekranAdlari[m.ekran]...)
}

func (m model) baslikSag() string {
	switch {
	case projeEkraniMi(m.ekran) && m.aktifProje != nil:
		s := ui.S.Accent.Render("⎇ " + safe(m.aktifProje.AktifDal))
		if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID && !m.ozet.durum.temizMi() {
			s += ui.S.Warning.Render(" ●")
		}
		return s
	case grupEkraniMi(m.ekran) && m.aktifGrup != nil:
		n := m.cevrimiciUyeSayisi(*m.aktifGrup)
		return ui.Dot(n > 0) + ui.S.Muted.Render(fmt.Sprintf(" %d/%d çevrimiçi", n, len(m.aktifGrup.Uyeler)-1))
	}
	return ui.S.Name.Render(safe(m.identity.Name))
}

// durumSatiri, alt bilgideki tek satırlık durumdur: süren işlem ya da son bildirim.
func (m model) durumSatiri() string {
	switch {
	case m.pending:
		return ui.S.Accent.Render(m.spinner.View()) + ui.S.Muted.Render(" İşleniyor…")
	case m.notice == "":
		return ""
	case m.isErr:
		return ui.Notice(ui.KindError, m.notice)
	}
	return ui.Notice(ui.KindSuccess, m.notice)
}

// ─── Yardımcılar ────────────────────────────────────────────────────────────

// sutun, s'yi (ekrandaki genişliğiyle) w sütuna boşlukla tamamlar: liste
// satırlarında adlardan sonraki bilgiler alt alta hizalansın.
func sutun(s string, w int) string {
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

// enGenis, adların en geniş olanının ekran genişliğidir (en fazla sinir).
func enGenis(adlar []string, sinir int) int {
	n := 0
	for _, a := range adlar {
		n = max(n, lipgloss.Width(a))
	}
	return min(n, sinir)
}

func kisa7(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

func kisaPeer(p string) string {
	if len(p) > 14 {
		return p[:12] + "…"
	}
	return p
}

// pencereli, satırları en fazla h satıra sığdırır: seçili satır görünür
// kalır, kesilen kısımlar "↑ N daha" / "↓ N daha" ile belirtilir.
func pencereli(satirlar []string, secili, h int) string {
	n := len(satirlar)
	if n <= h || h < 3 {
		return strings.Join(satirlar, "\n")
	}
	g := h - 2
	bas := max(0, min(secili-g/2, n-g))
	son := bas + g
	ust, alt := "", ""
	if bas > 0 {
		ust = ui.S.Subtle.Render(fmt.Sprintf("  ↑ %d daha", bas))
	}
	if son < n {
		alt = ui.S.Subtle.Render(fmt.Sprintf("  ↓ %d daha", n-son))
	}
	return ust + "\n" + strings.Join(satirlar[bas:son], "\n") + "\n" + alt
}

func (m model) cevrimiciUyeSayisi(g Grup) int {
	n := 0
	for _, u := range g.Uyeler {
		if u.PeerID != m.identity.PeerID && m.cevrimici[u.PeerID] {
			n++
		}
	}
	return n
}

// uyeAdi, aktif grupta peerID'nin adıdır; bilinmiyorsa kısaltılmış Peer ID.
func (m model) uyeAdi(peerID string) string {
	if m.aktifGrup != nil {
		for _, u := range m.aktifGrup.Uyeler {
			if u.PeerID == peerID && u.Ad != "" {
				return safe(u.Ad)
			}
		}
	}
	return kisaPeer(peerID)
}

func rol(g Grup, peerID string) string {
	switch {
	case g.benKurucuMuyum(peerID):
		return "kurucu"
	case g.yoneticiMi(peerID):
		return "yönetici"
	}
	return "üye"
}

// kayitOzeti, bir kaydı tek satırda yazar: kısa hash, mesaj, yazar ve zaman.
func (m model) kayitOzeti(k LogGirdisi) []ui.Seg {
	segs := []ui.Seg{ui.C(kisa7(k.Hash), ui.P.Accent), ui.T("  " + safe(ilkSatir(k.Commit.Mesaj)))}
	if len(k.Commit.Ebeveynler) > 1 {
		segs = append(segs, ui.C("  birleştirme", ui.P.Badge))
	}
	return append(segs, ui.M("  "+safe(k.Commit.Yazar)+" · "+goreliZaman(k.Commit.TS, simdi())))
}

func ilkSatir(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// degisiklikSeg, bir dosya değişikliğinin renkli işaretidir.
func degisiklikSeg(tip byte) ui.Seg {
	switch tip {
	case '+':
		return ui.Seg{Text: "+", Color: ui.P.Success, Bold: true, Fixed: true}
	case '-':
		return ui.Seg{Text: "−", Color: ui.P.Error, Bold: true, Fixed: true}
	}
	return ui.Seg{Text: "~", Color: ui.P.Warning, Bold: true, Fixed: true}
}

func degisiklikSayilari(d DurumSonucu) string {
	var p []string
	if n := len(d.Degisen); n > 0 {
		p = append(p, fmt.Sprintf("~%d değişen", n))
	}
	if n := len(d.Eklenen); n > 0 {
		p = append(p, fmt.Sprintf("+%d yeni", n))
	}
	if n := len(d.Silinen); n > 0 {
		p = append(p, fmt.Sprintf("−%d silinen", n))
	}
	return strings.Join(p, "  ")
}

// ikiSutun, sol ve sağ blokları aralarında ince bir çizgiyle yan yana koyar.
func ikiSutun(sol, sag string, solW, sagW int) string {
	h := max(strings.Count(sol, "\n"), strings.Count(sag, "\n")) + 1
	ayrac := strings.TrimSuffix(strings.Repeat(" "+ui.S.Subtle.Render("│")+" \n", h), "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top, ui.Block(sol, solW, h), ayrac, ui.Block(sag, sagW, h))
}

func (m model) viewGirdi(aciklama, girdi string, ipuclari ...ui.Hint) (string, []ui.Hint) {
	return ui.S.Muted.Render(ui.Wrap(aciklama, ui.Measure(m.icW()))) + "\n\n" + stBorder.Render(girdi) + "\n", ipuclari
}

// ─── Ana menü ───────────────────────────────────────────────────────────────

func (m model) anaMenuSatirlari() [][2]string {
	davet := "bekleyen davet yok"
	if n := len(m.bekleyenDavetler); n > 0 {
		davet = fmt.Sprintf("%d davet seni bekliyor", n)
	}
	return [][2]string{
		{"Gruplarım", fmt.Sprintf("%d grup", len(m.gruplar))},
		{"Yeni Grup Oluştur", "özel bir ağ kur, üyeleri davet et"},
		{"Davetler", davet},
		{"Kimlik Bilgileri", "Peer ID'ni paylaş"},
		{"Çıkış", ""},
	}
}

func (m model) viewAnaMenu(w int) (string, []ui.Hint) {
	ipuclari := []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "aç"), ui.H("q", "çıkış")}
	sagW := 0
	if w >= 96 {
		sagW = w * 42 / 100
	}
	solW := w
	if sagW > 0 {
		solW = w - sagW - 3
	}

	var sol strings.Builder
	sol.WriteString(ui.S.Muted.Render(ui.Wrap("Gruplar kur, kendi sürüm kontrol sistemiyle özel bir ağda birlikte proje geliştir.", solW)) + "\n\n")
	for i, s := range m.anaMenuSatirlari() {
		aciklama := ui.M("  " + s[1])
		if i == 2 && len(m.bekleyenDavetler) > 0 {
			aciklama = ui.Seg{Text: "  " + s[1], Color: ui.P.Warning}
		}
		sol.WriteString(ui.Row(solW, i == m.menuIdx, ui.Seg{Text: fmt.Sprintf("%-20s", s[0]), Bold: true}, aciklama) + "\n")
	}

	var sag strings.Builder
	sag.WriteString(ui.Section("Özet") + "\n")
	cevrimici := map[string]bool{}
	for _, g := range m.gruplar {
		for _, u := range g.Uyeler {
			if u.PeerID != m.identity.PeerID && m.cevrimici[u.PeerID] {
				cevrimici[u.PeerID] = true
			}
		}
	}
	sag.WriteString(ui.KV(12, "Gruplar", fmt.Sprint(len(m.gruplar))) + "\n")
	sag.WriteString(ui.KV(12, "Çevrimiçi", ui.Dot(len(cevrimici) > 0)+fmt.Sprintf(" %d grup arkadaşı", len(cevrimici))) + "\n")
	sag.WriteString(ui.KV(12, "Davetler", fmt.Sprint(len(m.bekleyenDavetler))) + "\n\n")
	sag.WriteString(ui.Section("Son kayıtlar") + "\n")
	if len(m.sonKayitlar) == 0 {
		sag.WriteString(ui.S.Muted.Render("Henüz kayıt yok.") + "\n")
	}
	genislik := w
	if sagW > 0 {
		genislik = sagW
	}
	for _, k := range m.sonKayitlar {
		sag.WriteString(ui.Truncate(ui.S.Accent.Render(kisa7(k.Hash))+" "+safe(ilkSatir(k.Commit.Mesaj)), genislik) + "\n")
		sag.WriteString(ui.Truncate(ui.S.Muted.Render("  "+safe(k.Commit.Yazar)+" · "+safe(k.proje)+" · "+goreliZaman(k.Commit.TS, simdi())), genislik) + "\n")
	}

	if sagW == 0 {
		return sol.String() + "\n" + sag.String(), ipuclari
	}
	return ikiSutun(sol.String(), sag.String(), solW, sagW) + "\n", ipuclari
}

func (m model) viewKimlik(w int) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.KV(14, "Ad", ui.S.Name.Render(safe(m.identity.Name))) + "\n")
	b.WriteString(ui.KV(14, "Peer ID", ui.Truncate(safe(m.identity.PeerID), w-14)) + "\n")
	b.WriteString(ui.KV(14, "Genel anahtar", ui.Truncate(safe(m.identity.PubKey), w-14)) + "\n\n")
	b.WriteString(ui.S.Muted.Render(ui.Wrap("Bir gruba davet edilmek için Peer ID'ni grubun kurucusuna ya da bir yöneticisine ilet (forum, sohbet ya da başka bir yoldan).", ui.Measure(w))) + "\n")
	return b.String(), []ui.Hint{ui.H("esc", "geri")}
}

// ─── Gruplar ────────────────────────────────────────────────────────────────

func (m model) viewGrupListesi(w int) (string, []ui.Hint) {
	var b strings.Builder
	if m.grupAramaAktif {
		b.WriteString(stBorder.Render(m.grupAramaIn.View()) + "\n\n")
	}
	gorunur := m.gruplarGorunur()
	switch {
	case len(m.gruplar) == 0:
		b.WriteString(ui.S.Muted.Render("Henüz bir gruba üye değilsin. Ana menüden \"Yeni Grup Oluştur\" ile başlayabilir ya da bir davet bekleyebilirsin.") + "\n")
	case len(gorunur) == 0:
		b.WriteString(ui.S.Muted.Render("Aramayla eşleşen grup yok.") + "\n")
	}
	var satirlar []string
	for i, g := range gorunur {
		segs := []ui.Seg{
			{Text: safe(g.Ad), Bold: true},
			ui.M(fmt.Sprintf("  %d üye · %s", len(g.Uyeler), rol(g, m.identity.PeerID))),
		}
		if n := m.cevrimiciUyeSayisi(g); n > 0 {
			segs = append(segs, ui.C(fmt.Sprintf("  ● %d çevrimiçi", n), ui.P.Success))
		}
		satirlar = append(satirlar, ui.Row(w, i == m.grupIdx, segs...))
	}
	b.WriteString(pencereli(satirlar, m.grupIdx, m.govdeYuksekligi()-2) + "\n")
	if m.grupAramaAktif {
		return b.String(), []ui.Hint{ui.H("yaz", "süz"), ui.H("enter", "aç"), ui.H("esc", "aramadan çık")}
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "aç"), ui.H("/", "ara"), ui.H("r", "yenile"), ui.H("esc", "geri")}
}

func (m model) viewGrupPaneli(w int) (string, []ui.Hint) {
	g := m.aktifGrup
	sagW := 0
	if w >= 90 {
		sagW = w * 38 / 100
	}
	solW := w
	if sagW > 0 {
		solW = w - sagW - 3
	}

	var sol strings.Builder
	sol.WriteString(ui.Section(fmt.Sprintf("Projeler (%d)", len(m.projeler))) + "\n")
	if len(m.projeler) == 0 {
		sol.WriteString(ui.S.Muted.Render("Henüz proje yok — n ile oluştur.") + "\n")
	}
	var satirlar []string
	for i, p := range m.projeler {
		segs := []ui.Seg{{Text: safe(p.Ad), Bold: true}, ui.C("  ⎇ "+safe(p.AktifDal), ui.P.Accent)}
		if k, ok := m.projeSonKayit[p.ID]; ok {
			segs = append(segs, ui.M("  · "+safe(ilkSatir(k.Commit.Mesaj))+" · "+goreliZaman(k.Commit.TS, simdi())))
		} else {
			segs = append(segs, ui.M("  · henüz kayıt yok"))
		}
		satirlar = append(satirlar, ui.Row(solW, i == m.projeIdx, segs...))
	}
	sol.WriteString(pencereli(satirlar, m.projeIdx, m.govdeYuksekligi()-2) + "\n")

	var sag strings.Builder
	sag.WriteString(ui.Section(fmt.Sprintf("Üyeler (%d)", len(g.Uyeler))) + "\n")
	uyeler := append([]Uye(nil), g.Uyeler...)
	sort.SliceStable(uyeler, func(i, j int) bool { return rolSirasi(*g, uyeler[i].PeerID) < rolSirasi(*g, uyeler[j].PeerID) })
	for _, u := range uyeler {
		ben := u.PeerID == m.identity.PeerID
		nokta := ui.Dot(ben || m.cevrimici[u.PeerID])
		ad := ui.S.Name.Render(safe(u.Ad))
		if ben {
			ad += ui.S.Muted.Render(" (sen)")
		}
		etiket := ""
		if r := rol(*g, u.PeerID); r != "üye" {
			etiket = "  " + ui.S.Badge.Render(r)
		}
		sag.WriteString(ui.Truncate(nokta+" "+ad+etiket, max(sagW, 20)) + "\n")
	}

	ipuclari := []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "projeyi aç"), ui.H("n", "yeni proje")}
	if g.yetkiliMi(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("i", "üye davet et"))
	}
	if g.benKurucuMuyum(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("y", "yöneticiler"))
	}
	if g.yetkiliMi(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("x", "grubu kapat"))
	} else {
		ipuclari = append(ipuclari, ui.H("x", "gruptan çık"))
	}
	ipuclari = append(ipuclari, ui.H("r", "yenile"), ui.H("esc", "geri"))

	if sagW == 0 {
		return sol.String() + "\n" + sag.String(), ipuclari
	}
	return ikiSutun(sol.String(), sag.String(), solW, sagW) + "\n", ipuclari
}

func rolSirasi(g Grup, peerID string) int {
	switch rol(g, peerID) {
	case "kurucu":
		return 0
	case "yönetici":
		return 1
	}
	return 2
}

func (m model) viewUyeDavet(w int) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(ui.Wrap("Davet edeceğin kişinin Peer ID'sini yaz (kendi ID'sini AND'ın Kimlik ekranından paylaşabilir). Kişi çevrimdışıysa AND daveti bekletir ve bağlandığında iletir.", ui.Measure(w))) + "\n\n")
	b.WriteString(stBorder.Render(m.uyeDavetIn.View()) + "\n\n")
	oneriler := m.davetOnerileri()
	b.WriteString(ui.Section("Bağlı AND düğümleri") + "\n")
	if len(oneriler) == 0 {
		b.WriteString(ui.S.Muted.Render("Şu an davet edilebilecek bağlı bir AND düğümü yok; Peer ID'yi elle yaz.") + "\n")
	}
	for i, p := range oneriler {
		b.WriteString(ui.Row(w, i == m.oneriIdx, ui.T(safe(p.PeerID))) + "\n")
	}
	ipuclari := []ui.Hint{ui.H("enter", "davet et"), ui.H("esc", "vazgeç")}
	if len(oneriler) > 0 {
		ipuclari = append([]ui.Hint{ui.H("↑↓", "öneriden seç")}, ipuclari...)
	}
	return b.String(), ipuclari
}

func (m model) viewDavetler(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.bekleyenDavetler) == 0 {
		b.WriteString(ui.S.Muted.Render("Bekleyen davet yok. Bir grubun kurucusu ya da yöneticisi seni davet ettiğinde burada görünür.") + "\n")
		return b.String(), []ui.Hint{ui.H("esc", "geri")}
	}
	for i, d := range m.bekleyenDavetler {
		b.WriteString(ui.Row(w, i == m.davetIdx,
			ui.Seg{Text: safe(d.Payload.GrupAd), Bold: true},
			ui.M(fmt.Sprintf("  davet eden: %s (%s) · %d üye", safe(d.Payload.Gonderen.Ad), kisaPeer(d.GonderenPeer), len(d.Payload.Uyeler))),
		) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("a", "kabul et"), ui.H("r", "reddet"), ui.H("esc", "geri")}
}

func (m model) viewYoneticiSecim(w int) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(ui.Wrap("Yöneticiler, kurucu gibi üye davet edebilir ve grubu kapatabilir: kurucuya ulaşılamazsa grup yönetimsiz kalmaz.", ui.Measure(w))) + "\n\n")
	if len(m.yoneticiHar) == 0 {
		b.WriteString(ui.S.Muted.Render("Grupta başka üye yok.") + "\n")
	}
	for i, u := range m.yoneticiHar {
		segs := []ui.Seg{{Text: safe(u.Ad), Bold: true}}
		if m.aktifGrup.yoneticiMi(u.PeerID) {
			segs = append(segs, ui.C("  yönetici", ui.P.Badge))
		}
		b.WriteString(ui.Row(w, i == m.yoneticiIdx, segs...) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "yönetici yap/kaldır"), ui.H("esc", "geri")}
}

// ─── Proje paneli ───────────────────────────────────────────────────────────

func (m model) viewProjePaneli(w int) (string, []ui.Hint) {
	p := m.aktifProje
	oz := m.ozet
	if oz != nil && oz.projeID != p.ID {
		oz = nil
	}
	const etW = 12
	var b strings.Builder

	son := ui.S.Muted.Render("hesaplanıyor…")
	calisma := son
	ekip := ""
	if oz != nil {
		switch {
		case oz.son != nil:
			son = ui.Truncate(ui.S.Accent.Render(kisa7(oz.son.Hash))+" "+safe(ilkSatir(oz.son.Commit.Mesaj))+
				ui.S.Muted.Render(" — "+safe(oz.son.Commit.Yazar)+", "+goreliZaman(oz.son.Commit.TS, simdi())), w-etW)
		default:
			son = ui.S.Muted.Render("henüz kayıt yok — c ile ilk kaydı oluştur")
		}
		if oz.durum.temizMi() {
			calisma = ui.S.Success.Render("✓ temiz")
		} else {
			calisma = ui.S.Warning.Render(degisiklikSayilari(oz.durum))
		}
		yeniUc := 0
		for _, n := range oz.uzakYeni {
			if n > 0 {
				yeniUc++
			}
		}
		if yeniUc > 0 {
			ekip = ui.S.Warning.Render(fmt.Sprintf("  · %d uçta birleştirilmemiş kayıt", yeniUc))
		}
	}
	b.WriteString(ui.KV(etW, "Dal", ui.S.Accent.Render("⎇ "+safe(p.AktifDal))) + ui.S.Muted.Render(fmt.Sprintf("  (%d dal)", len(p.Dallar))) + "\n")
	b.WriteString(ui.KV(etW, "Son kayıt", son) + "\n")
	b.WriteString(ui.KV(etW, "Çalışma", calisma) + "\n")
	b.WriteString(ui.KV(etW, "Ekip", fmt.Sprintf("%d/%d çevrimiçi", m.cevrimiciUyeSayisi(*m.aktifGrup), len(m.aktifGrup.Uyeler)-1)) + ekip + "\n")
	b.WriteString(ui.KV(etW, "Klasör", ui.S.Muted.Render(ui.Truncate(m.calismaDir(), w-etW))) + "\n")
	if _, birlesmeVar := birlesmeBekleniyorMu(m.dataDir, p.ID); birlesmeVar {
		b.WriteString("\n" + ui.Notice(ui.KindWarning, "Süren bir birleştirme var: çakışan dosyaları düzenleyip kaydet (c).") + "\n")
	}
	if oz != nil && oz.hata != "" {
		b.WriteString("\n" + ui.Notice(ui.KindError, oz.hata) + "\n")
	}
	b.WriteString("\n" + ui.Section("İşlemler") + "\n")
	for i, is := range projeIslemleri {
		segs := []ui.Seg{{Text: is.tus, Color: ui.P.Accent, Bold: true}, ui.T(fmt.Sprintf("  %-25s", is.ad))}
		switch {
		case is.tus == "d" && oz != nil && !oz.durum.temizMi():
			segs = append(segs, ui.C(fmt.Sprintf("%d dosya  ", len(degisiklikListesi(oz.durum))), ui.P.Warning))
		case is.tus == "m" && oz != nil:
			if n := toplamYeni(oz.uzakYeni); n > 0 {
				segs = append(segs, ui.C(fmt.Sprintf("%d yeni kayıt  ", n), ui.P.Warning))
			}
		}
		segs = append(segs, ui.M(is.aciklama))
		b.WriteString(ui.Row(w, i == m.projePanelIdx, segs...) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "çalıştır"), ui.H("d c l b p m u f", "doğrudan"), ui.H("esc", "geri")}
}

func toplamYeni(uzakYeni map[string]int) int {
	n := 0
	for _, v := range uzakYeni {
		n += v
	}
	return n
}

// ─── Değişiklikler ──────────────────────────────────────────────────────────

func (m model) viewDurum(w int) (string, []ui.Hint) {
	liste := degisiklikListesi(m.durum)
	if len(liste) == 0 {
		return ui.Notice(ui.KindSuccess, "Çalışma klasörü temiz: kaydedilmemiş değişiklik yok.") + "\n",
			[]ui.Hint{ui.H("r", "yenile"), ui.H("esc", "geri")}
	}
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(degisiklikSayilari(m.durum)+" — son kayda göre") + "\n\n")
	satirlar := make([]string, len(liste))
	for i, d := range liste {
		satirlar[i] = ui.Row(w, i == m.durumIdx, degisiklikSeg(d.Tip), ui.T("  "+safe(d.Yol)))
	}
	b.WriteString(pencereli(satirlar, m.durumIdx, m.govdeYuksekligi()-2) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "fark"), ui.H("g", "geri al"), ui.H("c", "kaydet"), ui.H("r", "yenile"), ui.H("esc", "geri")}
}

func (m model) viewCommitMesaj() (string, []ui.Hint) {
	var b strings.Builder
	if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID && !m.ozet.durum.temizMi() {
		b.WriteString(ui.S.Muted.Render("Kaydedilecek: ") + ui.S.Warning.Render(degisiklikSayilari(m.ozet.durum)) +
			ui.S.Muted.Render("  →  ⎇ "+safe(m.aktifProje.AktifDal)) + "\n\n")
	} else {
		b.WriteString(ui.S.Muted.Render("Çalışma klasörünün anlık görüntüsü ⎇ "+safe(m.aktifProje.AktifDal)+" dalına kaydedilir.") + "\n\n")
	}
	b.WriteString(stBorder.Render(m.commitMsgIn.View()) + "\n")
	return b.String(), []ui.Hint{ui.H("enter", "kaydet"), ui.H("esc", "vazgeç")}
}

// ─── Günlük ve kayıt ayrıntısı ──────────────────────────────────────────────

func (m model) viewGunluk(w int) (string, []ui.Hint) {
	var b strings.Builder
	gorunur := m.gunlukGorunur()
	ust := 2
	if m.gunlukAramaAktif || m.gunlukAramaIn.Value() != "" {
		b.WriteString(stBorder.Render(m.gunlukAramaIn.View()) + "\n")
		ust += 3
	}
	sayi := fmt.Sprintf("%d kayıt", len(gorunur))
	if len(gorunur) != len(m.gunluk) {
		sayi = fmt.Sprintf("%d / %d kayıt", len(gorunur), len(m.gunluk))
	}
	b.WriteString(ui.S.Muted.Render(sayi+" · ⎇ "+safe(m.aktifProje.AktifDal)) + "\n\n")
	if len(m.gunluk) == 0 {
		b.WriteString(ui.S.Muted.Render("Henüz kayıt yok. Proje panelinde c ile ilk kaydı oluştur.") + "\n")
	}
	satirlar := make([]string, len(gorunur))
	for i, k := range gorunur {
		satirlar[i] = ui.Row(w, i == m.gunlukIdx, m.kayitOzeti(k)...)
	}
	b.WriteString(pencereli(satirlar, m.gunlukIdx, m.govdeYuksekligi()-ust) + "\n")
	if m.gunlukAramaAktif {
		return b.String(), []ui.Hint{ui.H("yaz", "süz"), ui.H("↑↓", "seç"), ui.H("enter", "ayrıntı"), ui.H("esc", "aramadan çık")}
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "ayrıntı"), ui.H("/", "ara"), ui.H("esc", "geri")}
}

func (m model) viewKayit(w int) (string, []ui.Hint) {
	k := m.kayit
	c := k.commit
	const etW = 10
	var b strings.Builder
	b.WriteString(ui.KV(etW, "Kayıt", ui.S.Accent.Render(ui.Truncate(k.hash, w-etW))) + "\n")
	b.WriteString(ui.KV(etW, "Yazar", ui.S.Name.Render(safe(c.Yazar))) + "\n")
	b.WriteString(ui.KV(etW, "Tarih", c.TS.Local().Format("02.01.2006 15:04")+ui.S.Muted.Render("  ("+goreliZaman(c.TS, simdi())+")")) + "\n")
	switch len(c.Ebeveynler) {
	case 0:
		b.WriteString(ui.KV(etW, "Önceki", ui.S.Muted.Render("yok — projenin ilk kaydı")) + "\n")
	case 1:
		b.WriteString(ui.KV(etW, "Önceki", kisa7(c.Ebeveynler[0])) + "\n")
	default:
		var e []string
		for _, h := range c.Ebeveynler {
			e = append(e, kisa7(h))
		}
		b.WriteString(ui.KV(etW, "Önceki", strings.Join(e, " + ")+ui.S.Badge.Render("  birleştirme")) + "\n")
	}
	mesaj := ui.Wrap(safe(c.Mesaj), ui.Measure(w)-2)
	b.WriteString("\n" + ui.S.Card.Render(mesaj) + "\n\n")
	b.WriteString(ui.Section(fmt.Sprintf("Değişen dosyalar (%d)", len(k.degisiklikler))) + "\n")
	if len(k.degisiklikler) == 0 {
		b.WriteString(ui.S.Muted.Render("Bu kayıt dosya değiştirmiyor.") + "\n")
	}
	satirlar := make([]string, len(k.degisiklikler))
	for i, d := range k.degisiklikler {
		satirlar[i] = ui.Row(w, i == k.idx, degisiklikSeg(d.Tip), ui.T("  "+safe(d.Yol)))
	}
	kullanilan := 7 + strings.Count(mesaj, "\n") + 3
	b.WriteString(pencereli(satirlar, k.idx, m.govdeYuksekligi()-kullanilan) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "fark"), ui.H("b", "bu kayıttan dal"), ui.H("esc", "günlük")}
}

func (m model) viewDiff() (string, []ui.Hint) {
	ozet := ui.S.Success.Render(fmt.Sprintf("+%d", m.diffEklenen)) + " " + ui.S.Error.Render(fmt.Sprintf("−%d", m.diffSilinen)) + ui.S.Muted.Render(" satır")
	if m.diffGeri == ekKayit {
		ozet += ui.S.Muted.Render("  · " + kisa7(m.kayit.hash) + " kaydında, öncekine göre")
	} else {
		ozet += ui.S.Muted.Render("  · çalışma klasöründe, son kayda göre")
	}
	return ozet + "\n\n" + m.diffVP.View() + "\n", []ui.Hint{ui.H("↑↓", "kaydır"), ui.H("pgup/pgdn", "sayfa"), ui.H("esc", "geri")}
}

// ─── Dallar ─────────────────────────────────────────────────────────────────

func (m model) viewDallar(w int) (string, []ui.Hint) {
	var b strings.Builder
	satirlar := make([]string, len(m.dallar))
	var adlar []string
	for _, d := range m.dallar {
		adlar = append(adlar, safe(d))
	}
	adW := enGenis(adlar, 30)
	for i, d := range m.dallar {
		segs := []ui.Seg{{Text: sutun(safe(d), adW), Bold: true}}
		if d == m.aktifProje.AktifDal {
			segs = append(segs, ui.Seg{Text: "  ● etkin", Color: ui.P.Success, Fixed: true})
		} else {
			segs = append(segs, ui.T("         "))
		}
		if k, ok := m.dalSon[d]; ok {
			segs = append(segs, ui.M("  "+kisa7(k.Hash)+" "+safe(ilkSatir(k.Commit.Mesaj))+" · "+goreliZaman(k.Commit.TS, simdi())))
		} else {
			segs = append(segs, ui.M("  henüz kayıt yok"))
		}
		satirlar[i] = ui.Row(w, i == m.dalIdx, segs...)
	}
	b.WriteString(pencereli(satirlar, m.dalIdx, m.govdeYuksekligi()) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "dala geç"), ui.H("n", "yeni dal"), ui.H("d", "sil"), ui.H("esc", "geri")}
}

func (m model) viewYeniDal() (string, []ui.Hint) {
	kaynak := "⎇ " + safe(m.aktifProje.AktifDal) + " dalının son kaydından"
	if m.dalKaynak != "" {
		kaynak = kisa7(m.dalKaynak) + " kaydından"
	}
	return m.viewGirdi("Yeni dal "+kaynak+" başlar. Dala geçmek için Dallar listesinde enter'a bas.",
		m.yeniDalIn.View(), ui.H("enter", "oluştur"), ui.H("esc", "vazgeç"))
}

// ─── Ekip: gönder, çek, uzak dallar ─────────────────────────────────────────

func (m model) viewUyeSecimi(w, secili int, aciklama string, ipuclari ...ui.Hint) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(ui.Wrap(aciklama, ui.Measure(w))) + "\n\n")
	if len(m.uyelerHar) == 0 {
		b.WriteString(ui.S.Muted.Render("Grupta başka üye yok. Grup panelinde i ile üye davet edebilirsin.") + "\n")
		return b.String(), []ui.Hint{ui.H("esc", "geri")}
	}
	var adlar []string
	for _, u := range m.uyelerHar {
		adlar = append(adlar, safe(u.Ad))
	}
	adW := enGenis(adlar, 24)
	for i, u := range m.uyelerHar {
		durum := ui.M("  çevrimdışı")
		if m.cevrimici[u.PeerID] {
			durum = ui.Seg{Text: "  çevrimiçi ", Color: ui.P.Success, Fixed: true}
		}
		segs := []ui.Seg{{Text: sutun(safe(u.Ad), adW), Bold: true}, durum}
		if h := m.aktifProje.Uzaklar[u.PeerID+"/"+m.aktifProje.AktifDal]; h != "" {
			segs = append(segs, ui.M("  · bildiği son kayıt "+kisa7(h)))
		}
		b.WriteString(ui.Row(w, i == secili, segs...) + "\n")
	}
	return b.String(), ipuclari
}

func (m model) viewUzakDallar(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.uzakAnahtarlari) == 0 {
		b.WriteString(ui.S.Muted.Render(ui.Wrap("Henüz bir üyeden değişiklik gelmedi. Bir üyeden Çek İste (u) ya da üyelerin sana göndermesini bekle.", ui.Measure(w))) + "\n")
		return b.String(), []ui.Hint{ui.H("esc", "geri")}
	}
	b.WriteString(ui.S.Muted.Render(ui.Wrap("Üyelerden gelen dalların uçları. Birleştirme, seçilen ucu etkin dalına üç yönlü olarak katar.", ui.Measure(w))) + "\n\n")
	for i, k := range m.uzakAnahtarlari {
		peer, dal, _ := strings.Cut(k, "/")
		segs := []ui.Seg{{Text: m.uyeAdi(peer), Bold: true}, ui.C("  ⎇ "+safe(dal), ui.P.Accent), ui.M("  " + kisa7(m.aktifProje.Uzaklar[k]))}
		if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID {
			if n := m.ozet.uzakYeni[k]; n > 0 {
				segs = append(segs, ui.Seg{Text: fmt.Sprintf("  %d yeni kayıt", n), Color: ui.P.Warning, Fixed: true})
			} else {
				segs = append(segs, ui.Seg{Text: "  güncel", Color: ui.P.Success, Fixed: true})
			}
		}
		b.WriteString(ui.Row(w, i == m.uzakIdx, segs...) + "\n")
	}
	if len(m.sonCatismalar) > 0 {
		b.WriteString("\n" + ui.Notice(ui.KindWarning, "Son birleştirmede çakışan dosyalar:") + "\n")
		for _, c := range m.sonCatismalar {
			b.WriteString("  " + ui.S.Error.Render(safe(c)) + "\n")
		}
		b.WriteString(ui.S.Muted.Render(ui.Wrap("Bu dosyaları <<<<<<< / ======= / >>>>>>> işaretlerine göre düzenleyip kaydet (c).", ui.Measure(w))) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "birleştir"), ui.H("esc", "geri")}
}

// ─── Dosyalar ───────────────────────────────────────────────────────────────

func (m model) viewDosyaListesi(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.dosyaListesi) == 0 {
		b.WriteString(ui.S.Muted.Render("Çalışma klasöründe henüz dosya yok — n ile oluştur.") + "\n")
	}
	satirlar := make([]string, len(m.dosyaListesi))
	for i, d := range m.dosyaListesi {
		segs := []ui.Seg{ui.T(safe(d))}
		if d == m.dosyaSilOnayi {
			segs = append(segs, ui.Seg{Text: "  silmek için tekrar d", Color: ui.P.Error, Fixed: true})
		}
		satirlar[i] = ui.Row(w, i == m.dosyaIdx, segs...)
	}
	b.WriteString(pencereli(satirlar, m.dosyaIdx, m.govdeYuksekligi()) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", "seç"), ui.H("enter", "düzenle"), ui.H("n", "yeni dosya"), ui.H("d", "sil (iki kez)"), ui.H("esc", "geri")}
}

// ─── Diyaloglar ─────────────────────────────────────────────────────────────

// diyalog, içeriği ekranın ortasında kenarlıklı bir kutuda gösterir.
func (m model) diyalog(icerik string, kenar lipgloss.TerminalColor) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(kenar).Padding(1, 3).Render(icerik)
	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m model) viewDosyaOnay() string {
	if m.pendingConsent == nil {
		return ""
	}
	c := m.pendingConsent
	var b strings.Builder
	b.WriteString(ui.S.Warning.Bold(true).Render("Bilinmeyen gönderenden dosya") + "\n\n")
	b.WriteString(ui.KV(10, "Gönderen", ui.S.Muted.Render(kisaPeer(safe(c.SenderID)))) + "\n")
	b.WriteString(ui.KV(10, "Dosya", safe(c.Filename)) + "\n")
	b.WriteString(ui.KV(10, "Boyut", boyutYaz(c.Size)) + "\n\n")
	b.WriteString(ui.S.Muted.Render("Bu gönderen, bulunduğun hiçbir grubun üyesi değil.") + "\n\n")
	b.WriteString(ui.S.Key.Render("y") + " " + ui.S.KeyDesc.Render("kabul et") + ui.S.Subtle.Render(" · ") + ui.S.Key.Render("n") + " " + ui.S.KeyDesc.Render("reddet"))
	return m.diyalog(b.String(), ui.P.Warning)
}

func boyutYaz(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
}

// viewOnay, yıkıcı işlemler için ortak onay kutusudur (bkz. keyOnay).
func (m model) viewOnay() string {
	var baslik string
	var satirlar []string
	switch m.onayTip {
	case onayGrupCikKapat:
		if m.aktifGrup == nil {
			return ""
		}
		g := m.aktifGrup
		if g.yetkiliMi(m.identity.PeerID) {
			baslik = "Grubu kapat"
			satirlar = []string{fmt.Sprintf("\"%s\" grubunu kapatmak üzeresin.", safe(g.Ad)),
				"Tüm üyelere bildirilir ve grup herkesin listesinden kalkar."}
		} else {
			baslik = "Gruptan çık"
			satirlar = []string{fmt.Sprintf("\"%s\" grubundan çıkmak üzeresin.", safe(g.Ad)),
				"Kurucuya ve yöneticilere bildirilir, üye listesinden çıkarılırsın."}
		}
		satirlar = append(satirlar, "Proje dosyaların diskte kalır, kaybolmaz.")
	case onayDalDegistir:
		hedef := ""
		if m.dalIdx < len(m.dallar) {
			hedef = m.dallar[m.dalIdx]
		}
		baslik = "Dal değiştir"
		satirlar = []string{fmt.Sprintf("⎇ %s dalına geçmek üzeresin.", safe(hedef)),
			"Çalışma klasöründeki İZLENMEYEN ve .projeagignore ile YOK SAYILAN",
			"dosyalar dahil her şey silinip hedef dalın içeriğiyle değiştirilir."}
	case onayBirlestir:
		anahtar := ""
		if m.uzakIdx < len(m.uzakAnahtarlari) {
			anahtar = m.uzakAnahtarlari[m.uzakIdx]
		}
		peer, dal, _ := strings.Cut(anahtar, "/")
		baslik = "Uzak dalı birleştir"
		satirlar = []string{fmt.Sprintf("%s kişisinin ⎇ %s ucunu etkin dalınla birleştirmek üzeresin.", m.uyeAdi(peer), safe(dal)),
			"Çalışma klasöründeki İZLENMEYEN ve .projeagignore ile YOK SAYILAN",
			"dosyalar dahil her şey silinip birleştirme sonucuyla değiştirilir."}
	case onayGeriAl:
		baslik = "Değişikliği geri al"
		satirlar = []string{safe(m.geriAlYol),
			"Dosya son kayıttaki hâline döner; son kayıtta yoksa silinir.",
			"Kaydedilmemiş değişiklik kalıcı olarak kaybolur."}
	case onayDalSil:
		ad := ""
		if m.dalIdx < len(m.dallar) {
			ad = m.dallar[m.dalIdx]
		}
		baslik = "Dalı sil"
		satirlar = []string{fmt.Sprintf("⎇ %s dalını silmek üzeresin.", safe(ad)),
			"Kayıtlar geçmişte durur; yalnızca dalın adı kalkar."}
	}
	var b strings.Builder
	b.WriteString(ui.S.Warning.Bold(true).Render(baslik) + "\n\n")
	for i, s := range satirlar {
		if i == 0 {
			b.WriteString(s + "\n")
		} else {
			b.WriteString(ui.S.Muted.Render(s) + "\n")
		}
	}
	b.WriteString("\n")
	if m.pending {
		b.WriteString(ui.S.Accent.Render(m.spinner.View()) + ui.S.Muted.Render(" İşleniyor…") + "\n\n")
	}
	b.WriteString(ui.S.Key.Render("y") + " " + ui.S.KeyDesc.Render("onayla") + ui.S.Subtle.Render(" · ") + ui.S.Key.Render("n") + " " + ui.S.KeyDesc.Render("vazgeç"))
	return m.diyalog(b.String(), ui.P.Error)
}

// viewYardim, ? ile açılan kısayol kutusudur; içeriği önceki ekranınkidir.
func (m model) viewYardim() string {
	var b strings.Builder
	b.WriteString(ui.S.Title.Render("Kısayollar") + "\n\n")
	for _, k := range m.kisayollarIcin(m.yardimGeriEkran) {
		b.WriteString(ui.S.Key.Render(fmt.Sprintf("%-16s", k.tus)) + " " + k.aciklama + "\n")
	}
	b.WriteString("\n" + ui.S.Muted.Render("Her zaman: ctrl+c çıkış") + "\n\n")
	b.WriteString(ui.S.Subtle.Render("herhangi bir tuş — geri dön"))
	return m.diyalog(b.String(), ui.P.Accent)
}
