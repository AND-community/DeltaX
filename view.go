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

	"github.com/lucian95511/and/dil"
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
		ipuclari = append(ipuclari, ui.H("?", dil.T("yardım")))
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
		return m.viewGirdi(dil.T("Grup adını yaz. Grup için rastgele bir AES-256 anahtarı üretilir; yalnızca davet ettiğin üyeler bilir."),
			m.grupAdIn.View(), ui.H("enter", dil.T("oluştur")), ui.H("esc", dil.T("vazgeç")))
	case ekGrupPaneli:
		return m.viewGrupPaneli(w)
	case ekUyeDavet:
		return m.viewUyeDavet(w)
	case ekDavetler:
		return m.viewDavetler(w)
	case ekProjeOlustur:
		return m.viewGirdi(dil.T("Proje adını yaz. Proje bu grubun bütün üyeleriyle paylaşılabilir."),
			m.projeAdIn.View(), ui.H("enter", dil.T("oluştur")), ui.H("esc", dil.T("vazgeç")))
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
		return m.viewUyeSecimi(w, m.pushIdx, dil.T("Yeni kayıtlarını seçtiğin üyeye grup anahtarıyla şifreli bir paket olarak gönderir. Üyenin çevrimiçi olması gerekir."),
			ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("gönder")), ui.H("a", dil.T("çevrimiçi herkese")), ui.H("esc", dil.T("geri")))
	case ekUzakDallar:
		return m.viewUzakDallar(w)
	case ekPullIste:
		return m.viewUyeSecimi(w, m.pullIdx, dil.T("Seçtiğin üyeden kendi değişikliklerini sana göndermesini ister. Üyenin çevrimiçi olması gerekir."),
			ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("iste")), ui.H("a", dil.T("çevrimiçi herkesten")), ui.H("esc", dil.T("geri")))
	case ekYoneticiSecim:
		return m.viewYoneticiSecim(w)
	case ekDosyaListesi:
		return m.viewDosyaListesi(w)
	case ekYeniDosya:
		return m.viewGirdi(dil.T("Çalışma klasörüne göre dosya yolu (ör. src/ana.go). Klasörler gerekirse oluşturulur."),
			m.yeniDosyaIn.View(), ui.H("enter", dil.T("oluştur ve düzenle")), ui.H("esc", dil.T("vazgeç")))
	case ekDosyaDuzenle:
		return m.dosyaDuzenleTA.View() + "\n", []ui.Hint{ui.H("ctrl+s", dil.T("kaydet")), ui.H("esc", dil.T("listeye dön"))}
	case ekDisaAktar:
		return m.viewGirdi(dil.T("Projenin tüm geçmişi (nesneler ve çalışma klasörü) bu klasöre tek bir .zip dosyası olarak yazılır."),
			m.disaAktarIn.View(), ui.H("enter", dil.T("dışa aktar")), ui.H("esc", dil.T("vazgeç")))
	}
	return "", nil
}

// ─── Başlık ve alt bilgi ────────────────────────────────────────────────────

var ekranAdlari = map[ekran][]string{
	ekKimlik:        {dil.K("Kimlik")},
	ekGrupListesi:   {dil.K("Gruplarım")},
	ekGrupOlustur:   {dil.K("Yeni grup")},
	ekDavetler:      {dil.K("Davetler")},
	ekUyeDavet:      {dil.K("Üye davet et")},
	ekProjeOlustur:  {dil.K("Yeni proje")},
	ekYoneticiSecim: {dil.K("Yöneticiler")},
	ekDurum:         {dil.K("Değişiklikler")},
	ekCommitMesaj:   {dil.K("Kaydet")},
	ekGunluk:        {dil.K("Günlük")},
	ekDallar:        {dil.K("Dallar")},
	ekYeniDal:       {dil.K("Dallar"), dil.K("Yeni dal")},
	ekGonderSecim:   {dil.K("Gönder")},
	ekUzakDallar:    {dil.K("Uzak dallar")},
	ekPullIste:      {dil.K("Çek iste")},
	ekDosyaListesi:  {dil.K("Dosyalar")},
	ekYeniDosya:     {dil.K("Dosyalar"), dil.K("Yeni dosya")},
	ekDisaAktar:     {dil.K("Dışa aktar")},
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
	iz := []string{dil.T("DeltaX")}
	if (grupEkraniMi(m.ekran) || projeEkraniMi(m.ekran)) && m.aktifGrup != nil {
		iz = append(iz, safe(m.aktifGrup.Ad))
	}
	if projeEkraniMi(m.ekran) && m.aktifProje != nil {
		iz = append(iz, safe(m.aktifProje.Ad))
	}
	switch m.ekran {
	case ekKayit:
		return append(iz, dil.T("Günlük"), kisa7(m.kayit.hash))
	case ekDiff:
		if m.diffGeri == ekKayit {
			iz = append(iz, dil.T("Günlük"), kisa7(m.kayit.hash))
		} else {
			iz = append(iz, dil.T("Değişiklikler"))
		}
		return append(iz, safe(m.diffBaslik))
	case ekDosyaDuzenle:
		return append(iz, dil.T("Dosyalar"), safe(m.dosyaDuzenleYol))
	}
	for _, ad := range ekranAdlari[m.ekran] {
		iz = append(iz, dil.T(ad))
	}
	return iz
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
		return ui.Dot(n > 0) + ui.S.Muted.Render(dil.F(" %d/%d çevrimiçi", n, len(m.aktifGrup.Uyeler)-1))
	}
	return ui.S.Name.Render(safe(m.identity.Name))
}

// durumSatiri, alt bilgideki tek satırlık durumdur: süren işlem ya da son bildirim.
func (m model) durumSatiri() string {
	switch {
	case m.pending:
		return ui.S.Accent.Render(m.spinner.View()) + ui.S.Muted.Render(dil.T(" İşleniyor…"))
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

// etiketW, ui.KV satırlarının etiket sütunudur: en az enAz (Türkçe düzen),
// etkin dilde en uzun etiketten iki fazla.
func etiketW(enAz int, etiketler ...string) int {
	w := enAz
	for _, e := range etiketler {
		w = max(w, lipgloss.Width(dil.T(e))+2)
	}
	return w
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
		ust = ui.S.Subtle.Render(dil.F("  ↑ %d daha", bas))
	}
	if son < n {
		alt = ui.S.Subtle.Render(dil.F("  ↓ %d daha", n-son))
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
		return dil.K("kurucu")
	case g.yoneticiMi(peerID):
		return dil.K("yönetici")
	}
	return dil.K("üye")
}

// kayitOzeti, bir kaydı tek satırda yazar: kısa hash, mesaj, yazar ve zaman.
func (m model) kayitOzeti(k LogGirdisi) []ui.Seg {
	segs := []ui.Seg{ui.C(kisa7(k.Hash), ui.P.Accent), ui.T("  " + safe(ilkSatir(k.Commit.Mesaj)))}
	if len(k.Commit.Ebeveynler) > 1 {
		segs = append(segs, ui.C(dil.T("  birleştirme"), ui.P.Badge))
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
		p = append(p, dil.F("~%d değişen", n))
	}
	if n := len(d.Eklenen); n > 0 {
		p = append(p, dil.F("+%d yeni", n))
	}
	if n := len(d.Silinen); n > 0 {
		p = append(p, dil.F("−%d silinen", n))
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
	davet := dil.T("bekleyen davet yok")
	if n := len(m.bekleyenDavetler); n > 0 {
		davet = dil.F("%d davet seni bekliyor", n)
	}
	return [][2]string{
		{dil.T("Gruplarım"), dil.F("%d grup", len(m.gruplar))},
		{dil.T("Yeni Grup Oluştur"), dil.T("özel bir ağ kur, üyeleri davet et")},
		{dil.T("Davetler"), davet},
		{dil.T("Kimlik Bilgileri"), dil.T("Peer ID'ni paylaş")},
		{dil.T("Çıkış"), ""},
	}
}

func (m model) viewAnaMenu(w int) (string, []ui.Hint) {
	ipuclari := []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("aç")), ui.H("q", dil.T("çıkış"))}
	sagW := 0
	if w >= 96 {
		sagW = w * 42 / 100
	}
	solW := w
	if sagW > 0 {
		solW = w - sagW - 3
	}

	var sol strings.Builder
	sol.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Gruplar kur, kendi sürüm kontrol sistemiyle özel bir ağda birlikte proje geliştir."), solW)) + "\n\n")
	for i, s := range m.anaMenuSatirlari() {
		aciklama := ui.M("  " + s[1])
		if i == 2 && len(m.bekleyenDavetler) > 0 {
			aciklama = ui.Seg{Text: "  " + s[1], Color: ui.P.Warning}
		}
		sol.WriteString(ui.Row(solW, i == m.menuIdx, ui.Seg{Text: fmt.Sprintf("%-20s", s[0]), Bold: true}, aciklama) + "\n")
	}

	var sag strings.Builder
	sag.WriteString(ui.Section(dil.T("Özet")) + "\n")
	cevrimici := map[string]bool{}
	for _, g := range m.gruplar {
		for _, u := range g.Uyeler {
			if u.PeerID != m.identity.PeerID && m.cevrimici[u.PeerID] {
				cevrimici[u.PeerID] = true
			}
		}
	}
	ozetW := etiketW(12, "Gruplar", "Çevrimiçi", "Davetler")
	sag.WriteString(ui.KV(ozetW, dil.T("Gruplar"), fmt.Sprint(len(m.gruplar))) + "\n")
	sag.WriteString(ui.KV(ozetW, dil.T("Çevrimiçi"), ui.Dot(len(cevrimici) > 0)+dil.F(" %d grup arkadaşı", len(cevrimici))) + "\n")
	sag.WriteString(ui.KV(ozetW, dil.T("Davetler"), fmt.Sprint(len(m.bekleyenDavetler))) + "\n\n")
	sag.WriteString(ui.Section(dil.T("Son kayıtlar")) + "\n")
	if len(m.sonKayitlar) == 0 {
		sag.WriteString(ui.S.Muted.Render(dil.T("Henüz kayıt yok.")) + "\n")
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
	kimlikW := etiketW(14, "Ad", "Peer ID", "Genel anahtar")
	b.WriteString(ui.KV(kimlikW, dil.T("Ad"), ui.S.Name.Render(safe(m.identity.Name))) + "\n")
	b.WriteString(ui.KV(kimlikW, dil.T("Peer ID"), ui.Truncate(safe(m.identity.PeerID), w-kimlikW)) + "\n")
	b.WriteString(ui.KV(kimlikW, dil.T("Genel anahtar"), ui.Truncate(safe(m.identity.PubKey), w-kimlikW)) + "\n\n")
	b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Bir gruba davet edilmek için Peer ID'ni grubun kurucusuna ya da bir yöneticisine ilet (forum, sohbet ya da başka bir yoldan)."), ui.Measure(w))) + "\n")
	return b.String(), []ui.Hint{ui.H("esc", dil.T("geri"))}
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
		b.WriteString(ui.S.Muted.Render(dil.T("Henüz bir gruba üye değilsin. Ana menüden \"Yeni Grup Oluştur\" ile başlayabilir ya da bir davet bekleyebilirsin.")) + "\n")
	case len(gorunur) == 0:
		b.WriteString(ui.S.Muted.Render(dil.T("Aramayla eşleşen grup yok.")) + "\n")
	}
	var satirlar []string
	for i, g := range gorunur {
		segs := []ui.Seg{
			{Text: safe(g.Ad), Bold: true},
			ui.M(dil.F("  %d üye · %s", len(g.Uyeler), dil.T(rol(g, m.identity.PeerID)))),
		}
		if n := m.cevrimiciUyeSayisi(g); n > 0 {
			segs = append(segs, ui.C(dil.F("  ● %d çevrimiçi", n), ui.P.Success))
		}
		satirlar = append(satirlar, ui.Row(w, i == m.grupIdx, segs...))
	}
	b.WriteString(pencereli(satirlar, m.grupIdx, m.govdeYuksekligi()-2) + "\n")
	if m.grupAramaAktif {
		return b.String(), []ui.Hint{ui.H("yaz", dil.T("süz")), ui.H("enter", dil.T("aç")), ui.H("esc", dil.T("aramadan çık"))}
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("aç")), ui.H("/", dil.T("ara")), ui.H("r", dil.T("yenile")), ui.H("esc", dil.T("geri"))}
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
	sol.WriteString(ui.Section(dil.F("Projeler (%d)", len(m.projeler))) + "\n")
	if len(m.projeler) == 0 {
		sol.WriteString(ui.S.Muted.Render(dil.T("Henüz proje yok — n ile oluştur.")) + "\n")
	}
	var satirlar []string
	for i, p := range m.projeler {
		segs := []ui.Seg{{Text: safe(p.Ad), Bold: true}, ui.C("  ⎇ "+safe(p.AktifDal), ui.P.Accent)}
		if k, ok := m.projeSonKayit[p.ID]; ok {
			segs = append(segs, ui.M("  · "+safe(ilkSatir(k.Commit.Mesaj))+" · "+goreliZaman(k.Commit.TS, simdi())))
		} else {
			segs = append(segs, ui.M(dil.T("  · henüz kayıt yok")))
		}
		satirlar = append(satirlar, ui.Row(solW, i == m.projeIdx, segs...))
	}
	sol.WriteString(pencereli(satirlar, m.projeIdx, m.govdeYuksekligi()-2) + "\n")

	var sag strings.Builder
	sag.WriteString(ui.Section(dil.F("Üyeler (%d)", len(g.Uyeler))) + "\n")
	uyeler := append([]Uye(nil), g.Uyeler...)
	sort.SliceStable(uyeler, func(i, j int) bool { return rolSirasi(*g, uyeler[i].PeerID) < rolSirasi(*g, uyeler[j].PeerID) })
	for _, u := range uyeler {
		ben := u.PeerID == m.identity.PeerID
		nokta := ui.Dot(ben || m.cevrimici[u.PeerID])
		ad := ui.S.Name.Render(safe(u.Ad))
		if ben {
			ad += ui.S.Muted.Render(dil.T(" (sen)"))
		}
		etiket := ""
		if r := rol(*g, u.PeerID); r != "üye" {
			etiket = "  " + ui.S.Badge.Render(dil.T(r))
		}
		sag.WriteString(ui.Truncate(nokta+" "+ad+etiket, max(sagW, 20)) + "\n")
	}

	ipuclari := []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("projeyi aç")), ui.H("n", dil.T("yeni proje"))}
	if g.yetkiliMi(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("i", dil.T("üye davet et")))
	}
	if g.benKurucuMuyum(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("y", dil.T("yöneticiler")))
	}
	if g.yetkiliMi(m.identity.PeerID) {
		ipuclari = append(ipuclari, ui.H("x", dil.T("grubu kapat")))
	} else {
		ipuclari = append(ipuclari, ui.H("x", dil.T("gruptan çık")))
	}
	ipuclari = append(ipuclari, ui.H("r", dil.T("yenile")), ui.H("esc", dil.T("geri")))

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
	b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Davet edeceğin kişinin Peer ID'sini yaz (kendi ID'sini AND'ın Kimlik ekranından paylaşabilir). Kişi çevrimdışıysa AND daveti bekletir ve bağlandığında iletir."), ui.Measure(w))) + "\n\n")
	b.WriteString(stBorder.Render(m.uyeDavetIn.View()) + "\n\n")
	oneriler := m.davetOnerileri()
	b.WriteString(ui.Section(dil.T("Bağlı AND düğümleri")) + "\n")
	if len(oneriler) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Şu an davet edilebilecek bağlı bir AND düğümü yok; Peer ID'yi elle yaz.")) + "\n")
	}
	for i, p := range oneriler {
		b.WriteString(ui.Row(w, i == m.oneriIdx, ui.T(safe(p.PeerID))) + "\n")
	}
	ipuclari := []ui.Hint{ui.H("enter", dil.T("davet et")), ui.H("esc", dil.T("vazgeç"))}
	if len(oneriler) > 0 {
		ipuclari = append([]ui.Hint{ui.H("↑↓", dil.T("öneriden seç"))}, ipuclari...)
	}
	return b.String(), ipuclari
}

func (m model) viewDavetler(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.bekleyenDavetler) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Bekleyen davet yok. Bir grubun kurucusu ya da yöneticisi seni davet ettiğinde burada görünür.")) + "\n")
		return b.String(), []ui.Hint{ui.H("esc", dil.T("geri"))}
	}
	for i, d := range m.bekleyenDavetler {
		b.WriteString(ui.Row(w, i == m.davetIdx,
			ui.Seg{Text: safe(d.Payload.GrupAd), Bold: true},
			ui.M(dil.F("  davet eden: %s (%s) · %d üye", safe(d.Payload.Gonderen.Ad), kisaPeer(d.GonderenPeer), len(d.Payload.Uyeler))),
		) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("a", dil.T("kabul et")), ui.H("r", dil.T("reddet")), ui.H("esc", dil.T("geri"))}
}

func (m model) viewYoneticiSecim(w int) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Yöneticiler, kurucu gibi üye davet edebilir ve grubu kapatabilir: kurucuya ulaşılamazsa grup yönetimsiz kalmaz."), ui.Measure(w))) + "\n\n")
	if len(m.yoneticiHar) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Grupta başka üye yok.")) + "\n")
	}
	for i, u := range m.yoneticiHar {
		segs := []ui.Seg{{Text: safe(u.Ad), Bold: true}}
		if m.aktifGrup.yoneticiMi(u.PeerID) {
			segs = append(segs, ui.C(dil.T("  yönetici"), ui.P.Badge))
		}
		b.WriteString(ui.Row(w, i == m.yoneticiIdx, segs...) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("yönetici yap/kaldır")), ui.H("esc", dil.T("geri"))}
}

// ─── Proje paneli ───────────────────────────────────────────────────────────

func (m model) viewProjePaneli(w int) (string, []ui.Hint) {
	p := m.aktifProje
	oz := m.ozet
	if oz != nil && oz.projeID != p.ID {
		oz = nil
	}
	etW := etiketW(12, "Dal", "Son kayıt", "Çalışma", "Ekip", "Klasör")
	var b strings.Builder

	son := ui.S.Muted.Render(dil.T("hesaplanıyor…"))
	calisma := son
	ekip := ""
	if oz != nil {
		switch {
		case oz.son != nil:
			son = ui.Truncate(ui.S.Accent.Render(kisa7(oz.son.Hash))+" "+safe(ilkSatir(oz.son.Commit.Mesaj))+
				ui.S.Muted.Render(" — "+safe(oz.son.Commit.Yazar)+", "+goreliZaman(oz.son.Commit.TS, simdi())), w-etW)
		default:
			son = ui.S.Muted.Render(dil.T("henüz kayıt yok — c ile ilk kaydı oluştur"))
		}
		if oz.durum.temizMi() {
			calisma = ui.S.Success.Render(dil.T("✓ temiz"))
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
			ekip = ui.S.Warning.Render(dil.F("  · %d uçta birleştirilmemiş kayıt", yeniUc))
		}
	}
	b.WriteString(ui.KV(etW, dil.T("Dal"), ui.S.Accent.Render("⎇ "+safe(p.AktifDal))) + ui.S.Muted.Render(dil.F("  (%d dal)", len(p.Dallar))) + "\n")
	b.WriteString(ui.KV(etW, dil.T("Son kayıt"), son) + "\n")
	b.WriteString(ui.KV(etW, dil.T("Çalışma"), calisma) + "\n")
	b.WriteString(ui.KV(etW, dil.T("Ekip"), dil.F("%d/%d çevrimiçi", m.cevrimiciUyeSayisi(*m.aktifGrup), len(m.aktifGrup.Uyeler)-1)) + ekip + "\n")
	b.WriteString(ui.KV(etW, dil.T("Klasör"), ui.S.Muted.Render(ui.Truncate(m.calismaDir(), w-etW))) + "\n")
	if _, birlesmeVar := birlesmeBekleniyorMu(m.dataDir, p.ID); birlesmeVar {
		b.WriteString("\n" + ui.Notice(ui.KindWarning, dil.T("Süren bir birleştirme var: çakışan dosyaları düzenleyip kaydet (c).")) + "\n")
	}
	if oz != nil && oz.hata != "" {
		b.WriteString("\n" + ui.Notice(ui.KindError, oz.hata) + "\n")
	}
	b.WriteString("\n" + ui.Section(dil.T("İşlemler")) + "\n")
	// Ad sütunu etkin dildeki en uzun işlem adına göre (en az 25, açıklamayla
	// arasında en az iki boşluk) hizalanır.
	adW := 25
	for _, is := range projeIslemleri {
		adW = max(adW, lipgloss.Width(dil.T(is.ad))+2)
	}
	for i, is := range projeIslemleri {
		segs := []ui.Seg{{Text: is.tus, Color: ui.P.Accent, Bold: true}, ui.T("  " + sutun(dil.T(is.ad), adW))}
		switch {
		case is.tus == "d" && oz != nil && !oz.durum.temizMi():
			segs = append(segs, ui.C(dil.F("%d dosya  ", len(degisiklikListesi(oz.durum))), ui.P.Warning))
		case is.tus == "m" && oz != nil:
			if n := toplamYeni(oz.uzakYeni); n > 0 {
				segs = append(segs, ui.C(dil.F("%d yeni kayıt  ", n), ui.P.Warning))
			}
		}
		segs = append(segs, ui.M(dil.T(is.aciklama)))
		b.WriteString(ui.Row(w, i == m.projePanelIdx, segs...) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("çalıştır")), ui.H("d c l b p m u f", dil.T("doğrudan")), ui.H("esc", dil.T("geri"))}
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
		return ui.Notice(ui.KindSuccess, dil.T("Çalışma klasörü temiz: kaydedilmemiş değişiklik yok.")) + "\n",
			[]ui.Hint{ui.H("r", dil.T("yenile")), ui.H("esc", dil.T("geri"))}
	}
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(degisiklikSayilari(m.durum)+dil.T(" — son kayda göre")) + "\n\n")
	satirlar := make([]string, len(liste))
	for i, d := range liste {
		satirlar[i] = ui.Row(w, i == m.durumIdx, degisiklikSeg(d.Tip), ui.T("  "+safe(d.Yol)))
	}
	b.WriteString(pencereli(satirlar, m.durumIdx, m.govdeYuksekligi()-2) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("fark")), ui.H("g", dil.T("geri al")), ui.H("c", dil.T("kaydet")), ui.H("r", dil.T("yenile")), ui.H("esc", dil.T("geri"))}
}

func (m model) viewCommitMesaj() (string, []ui.Hint) {
	var b strings.Builder
	if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID && !m.ozet.durum.temizMi() {
		b.WriteString(ui.S.Muted.Render(dil.T("Kaydedilecek: ")) + ui.S.Warning.Render(degisiklikSayilari(m.ozet.durum)) +
			ui.S.Muted.Render("  →  ⎇ "+safe(m.aktifProje.AktifDal)) + "\n\n")
	} else {
		b.WriteString(ui.S.Muted.Render(dil.F("Çalışma klasörünün anlık görüntüsü ⎇ %s dalına kaydedilir.", safe(m.aktifProje.AktifDal))) + "\n\n")
	}
	b.WriteString(stBorder.Render(m.commitMsgIn.View()) + "\n")
	return b.String(), []ui.Hint{ui.H("enter", dil.T("kaydet")), ui.H("esc", dil.T("vazgeç"))}
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
	sayi := dil.F("%d kayıt", len(gorunur))
	if len(gorunur) != len(m.gunluk) {
		sayi = dil.F("%d / %d kayıt", len(gorunur), len(m.gunluk))
	}
	b.WriteString(ui.S.Muted.Render(sayi+" · ⎇ "+safe(m.aktifProje.AktifDal)) + "\n\n")
	if len(m.gunluk) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Henüz kayıt yok. Proje panelinde c ile ilk kaydı oluştur.")) + "\n")
	}
	satirlar := make([]string, len(gorunur))
	for i, k := range gorunur {
		satirlar[i] = ui.Row(w, i == m.gunlukIdx, m.kayitOzeti(k)...)
	}
	b.WriteString(pencereli(satirlar, m.gunlukIdx, m.govdeYuksekligi()-ust) + "\n")
	if m.gunlukAramaAktif {
		return b.String(), []ui.Hint{ui.H("yaz", dil.T("süz")), ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("ayrıntı")), ui.H("esc", dil.T("aramadan çık"))}
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("ayrıntı")), ui.H("/", dil.T("ara")), ui.H("esc", dil.T("geri"))}
}

func (m model) viewKayit(w int) (string, []ui.Hint) {
	k := m.kayit
	c := k.commit
	etW := etiketW(10, "Kayıt", "Yazar", "Tarih", "Önceki")
	var b strings.Builder
	b.WriteString(ui.KV(etW, dil.T("Kayıt"), ui.S.Accent.Render(ui.Truncate(k.hash, w-etW))) + "\n")
	b.WriteString(ui.KV(etW, dil.T("Yazar"), ui.S.Name.Render(safe(c.Yazar))) + "\n")
	b.WriteString(ui.KV(etW, dil.T("Tarih"), c.TS.Local().Format("02.01.2006 15:04")+ui.S.Muted.Render("  ("+goreliZaman(c.TS, simdi())+")")) + "\n")
	switch len(c.Ebeveynler) {
	case 0:
		b.WriteString(ui.KV(etW, dil.T("Önceki"), ui.S.Muted.Render(dil.T("yok — projenin ilk kaydı"))) + "\n")
	case 1:
		b.WriteString(ui.KV(etW, dil.T("Önceki"), kisa7(c.Ebeveynler[0])) + "\n")
	default:
		var e []string
		for _, h := range c.Ebeveynler {
			e = append(e, kisa7(h))
		}
		b.WriteString(ui.KV(etW, dil.T("Önceki"), strings.Join(e, " + ")+ui.S.Badge.Render(dil.T("  birleştirme"))) + "\n")
	}
	mesaj := ui.Wrap(safe(c.Mesaj), ui.Measure(w)-2)
	b.WriteString("\n" + ui.S.Card.Render(mesaj) + "\n\n")
	b.WriteString(ui.Section(dil.F("Değişen dosyalar (%d)", len(k.degisiklikler))) + "\n")
	if len(k.degisiklikler) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Bu kayıt dosya değiştirmiyor.")) + "\n")
	}
	satirlar := make([]string, len(k.degisiklikler))
	for i, d := range k.degisiklikler {
		satirlar[i] = ui.Row(w, i == k.idx, degisiklikSeg(d.Tip), ui.T("  "+safe(d.Yol)))
	}
	kullanilan := 7 + strings.Count(mesaj, "\n") + 3
	b.WriteString(pencereli(satirlar, k.idx, m.govdeYuksekligi()-kullanilan) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("fark")), ui.H("b", dil.T("bu kayıttan dal")), ui.H("esc", dil.T("günlük"))}
}

func (m model) viewDiff() (string, []ui.Hint) {
	ozet := ui.S.Success.Render(fmt.Sprintf("+%d", m.diffEklenen)) + " " + ui.S.Error.Render(fmt.Sprintf("−%d", m.diffSilinen)) + ui.S.Muted.Render(dil.T(" satır"))
	if m.diffGeri == ekKayit {
		ozet += ui.S.Muted.Render(dil.F("  · %s kaydında, öncekine göre", kisa7(m.kayit.hash)))
	} else {
		ozet += ui.S.Muted.Render(dil.T("  · çalışma klasöründe, son kayda göre"))
	}
	return ozet + "\n\n" + m.diffVP.View() + "\n", []ui.Hint{ui.H("↑↓", dil.T("kaydır")), ui.H("pgup/pgdn", dil.T("sayfa")), ui.H("esc", dil.T("geri"))}
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
			segs = append(segs, ui.Seg{Text: dil.T("  ● etkin"), Color: ui.P.Success, Fixed: true})
		} else {
			segs = append(segs, ui.T(strings.Repeat(" ", lipgloss.Width(dil.T("  ● etkin"))))) // etkin dalla hizalı
		}
		if k, ok := m.dalSon[d]; ok {
			segs = append(segs, ui.M("  "+kisa7(k.Hash)+" "+safe(ilkSatir(k.Commit.Mesaj))+" · "+goreliZaman(k.Commit.TS, simdi())))
		} else {
			segs = append(segs, ui.M(dil.T("  henüz kayıt yok")))
		}
		satirlar[i] = ui.Row(w, i == m.dalIdx, segs...)
	}
	b.WriteString(pencereli(satirlar, m.dalIdx, m.govdeYuksekligi()) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("dala geç")), ui.H("n", dil.T("yeni dal")), ui.H("d", dil.T("sil")), ui.H("esc", dil.T("geri"))}
}

func (m model) viewYeniDal() (string, []ui.Hint) {
	kaynak := dil.F("⎇ %s dalının son kaydından", safe(m.aktifProje.AktifDal))
	if m.dalKaynak != "" {
		kaynak = dil.F("%s kaydından", kisa7(m.dalKaynak))
	}
	return m.viewGirdi(dil.F("Yeni dal %s başlar. Dala geçmek için Dallar listesinde enter'a bas.", kaynak),
		m.yeniDalIn.View(), ui.H("enter", dil.T("oluştur")), ui.H("esc", dil.T("vazgeç")))
}

// ─── Ekip: gönder, çek, uzak dallar ─────────────────────────────────────────

func (m model) viewUyeSecimi(w, secili int, aciklama string, ipuclari ...ui.Hint) (string, []ui.Hint) {
	var b strings.Builder
	b.WriteString(ui.S.Muted.Render(ui.Wrap(aciklama, ui.Measure(w))) + "\n\n")
	if len(m.uyelerHar) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Grupta başka üye yok. Grup panelinde i ile üye davet edebilirsin.")) + "\n")
		return b.String(), []ui.Hint{ui.H("esc", dil.T("geri"))}
	}
	var adlar []string
	for _, u := range m.uyelerHar {
		adlar = append(adlar, safe(u.Ad))
	}
	adW := enGenis(adlar, 24)
	for i, u := range m.uyelerHar {
		cevrimW := max(lipgloss.Width(dil.T("  çevrimdışı")), lipgloss.Width(dil.T("  çevrimiçi ")))
		durum := ui.M(sutun(dil.T("  çevrimdışı"), cevrimW))
		if m.cevrimici[u.PeerID] {
			durum = ui.Seg{Text: sutun(dil.T("  çevrimiçi "), cevrimW), Color: ui.P.Success, Fixed: true}
		}
		segs := []ui.Seg{{Text: sutun(safe(u.Ad), adW), Bold: true}, durum}
		if h := m.aktifProje.Uzaklar[u.PeerID+"/"+m.aktifProje.AktifDal]; h != "" {
			segs = append(segs, ui.M(dil.T("  · bildiği son kayıt ")+kisa7(h)))
		}
		b.WriteString(ui.Row(w, i == secili, segs...) + "\n")
	}
	return b.String(), ipuclari
}

func (m model) viewUzakDallar(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.uzakAnahtarlari) == 0 {
		b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Henüz bir üyeden değişiklik gelmedi. Bir üyeden Çek İste (u) ya da üyelerin sana göndermesini bekle."), ui.Measure(w))) + "\n")
		return b.String(), []ui.Hint{ui.H("esc", dil.T("geri"))}
	}
	b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Üyelerden gelen dalların uçları. Birleştirme, seçilen ucu etkin dalına üç yönlü olarak katar."), ui.Measure(w))) + "\n\n")
	for i, k := range m.uzakAnahtarlari {
		peer, dal, _ := strings.Cut(k, "/")
		segs := []ui.Seg{{Text: m.uyeAdi(peer), Bold: true}, ui.C("  ⎇ "+safe(dal), ui.P.Accent), ui.M("  " + kisa7(m.aktifProje.Uzaklar[k]))}
		if m.ozet != nil && m.ozet.projeID == m.aktifProje.ID {
			if n := m.ozet.uzakYeni[k]; n > 0 {
				segs = append(segs, ui.Seg{Text: dil.F("  %d yeni kayıt", n), Color: ui.P.Warning, Fixed: true})
			} else {
				segs = append(segs, ui.Seg{Text: dil.T("  güncel"), Color: ui.P.Success, Fixed: true})
			}
		}
		b.WriteString(ui.Row(w, i == m.uzakIdx, segs...) + "\n")
	}
	if len(m.sonCatismalar) > 0 {
		b.WriteString("\n" + ui.Notice(ui.KindWarning, dil.T("Son birleştirmede çakışan dosyalar:")) + "\n")
		for _, c := range m.sonCatismalar {
			b.WriteString("  " + ui.S.Error.Render(safe(c)) + "\n")
		}
		b.WriteString(ui.S.Muted.Render(ui.Wrap(dil.T("Bu dosyaları <<<<<<< / ======= / >>>>>>> işaretlerine göre düzenleyip kaydet (c)."), ui.Measure(w))) + "\n")
	}
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("birleştir")), ui.H("esc", dil.T("geri"))}
}

// ─── Dosyalar ───────────────────────────────────────────────────────────────

func (m model) viewDosyaListesi(w int) (string, []ui.Hint) {
	var b strings.Builder
	if len(m.dosyaListesi) == 0 {
		b.WriteString(ui.S.Muted.Render(dil.T("Çalışma klasöründe henüz dosya yok — n ile oluştur.")) + "\n")
	}
	satirlar := make([]string, len(m.dosyaListesi))
	for i, d := range m.dosyaListesi {
		segs := []ui.Seg{ui.T(safe(d))}
		if d == m.dosyaSilOnayi {
			segs = append(segs, ui.Seg{Text: dil.T("  silmek için tekrar d"), Color: ui.P.Error, Fixed: true})
		}
		satirlar[i] = ui.Row(w, i == m.dosyaIdx, segs...)
	}
	b.WriteString(pencereli(satirlar, m.dosyaIdx, m.govdeYuksekligi()) + "\n")
	return b.String(), []ui.Hint{ui.H("↑↓", dil.T("seç")), ui.H("enter", dil.T("düzenle")), ui.H("n", dil.T("yeni dosya")), ui.H("d", dil.T("sil (iki kez)")), ui.H("esc", dil.T("geri"))}
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
	b.WriteString(ui.S.Warning.Bold(true).Render(dil.T("Bilinmeyen gönderenden dosya")) + "\n\n")
	gonderenW := etiketW(10, "Gönderen", "Dosya", "Boyut")
	b.WriteString(ui.KV(gonderenW, dil.T("Gönderen"), ui.S.Muted.Render(kisaPeer(safe(c.SenderID)))) + "\n")
	b.WriteString(ui.KV(gonderenW, dil.T("Dosya"), safe(c.Filename)) + "\n")
	b.WriteString(ui.KV(gonderenW, dil.T("Boyut"), boyutYaz(c.Size)) + "\n\n")
	b.WriteString(ui.S.Muted.Render(dil.T("Bu gönderen, bulunduğun hiçbir grubun üyesi değil.")) + "\n\n")
	b.WriteString(ui.S.Key.Render("y") + " " + ui.S.KeyDesc.Render(dil.T("kabul et")) + ui.S.Subtle.Render(" · ") + ui.S.Key.Render("n") + " " + ui.S.KeyDesc.Render(dil.T("reddet")))
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
			baslik = dil.T("Grubu kapat")
			satirlar = []string{dil.F("\"%s\" grubunu kapatmak üzeresin.", safe(g.Ad)),
				dil.T("Tüm üyelere bildirilir ve grup herkesin listesinden kalkar.")}
		} else {
			baslik = dil.T("Gruptan çık")
			satirlar = []string{dil.F("\"%s\" grubundan çıkmak üzeresin.", safe(g.Ad)),
				dil.T("Kurucuya ve yöneticilere bildirilir, üye listesinden çıkarılırsın.")}
		}
		satirlar = append(satirlar, dil.T("Proje dosyaların diskte kalır, kaybolmaz."))
	case onayDalDegistir:
		hedef := ""
		if m.dalIdx < len(m.dallar) {
			hedef = m.dallar[m.dalIdx]
		}
		baslik = dil.T("Dal değiştir")
		satirlar = []string{dil.F("⎇ %s dalına geçmek üzeresin.", safe(hedef)),
			dil.T("Çalışma klasöründeki İZLENMEYEN ve .projeagignore ile YOK SAYILAN"),
			dil.T("dosyalar dahil her şey silinip hedef dalın içeriğiyle değiştirilir.")}
	case onayBirlestir:
		anahtar := ""
		if m.uzakIdx < len(m.uzakAnahtarlari) {
			anahtar = m.uzakAnahtarlari[m.uzakIdx]
		}
		peer, dal, _ := strings.Cut(anahtar, "/")
		baslik = dil.T("Uzak dalı birleştir")
		satirlar = []string{dil.F("%s kişisinin ⎇ %s ucunu etkin dalınla birleştirmek üzeresin.", m.uyeAdi(peer), safe(dal)),
			dil.T("Çalışma klasöründeki İZLENMEYEN ve .projeagignore ile YOK SAYILAN"),
			dil.T("dosyalar dahil her şey silinip birleştirme sonucuyla değiştirilir.")}
	case onayGeriAl:
		baslik = dil.T("Değişikliği geri al")
		satirlar = []string{safe(m.geriAlYol),
			dil.T("Dosya son kayıttaki hâline döner; son kayıtta yoksa silinir."),
			dil.T("Kaydedilmemiş değişiklik kalıcı olarak kaybolur.")}
	case onayDalSil:
		ad := ""
		if m.dalIdx < len(m.dallar) {
			ad = m.dallar[m.dalIdx]
		}
		baslik = dil.T("Dalı sil")
		satirlar = []string{dil.F("⎇ %s dalını silmek üzeresin.", safe(ad)),
			dil.T("Kayıtlar geçmişte durur; yalnızca dalın adı kalkar.")}
	}
	var b strings.Builder
	b.WriteString(ui.S.Warning.Bold(true).Render(baslik) + "\n\n")
	// Kutu dar terminalden taşmasın diye satırlar kutunun iç genişliğinde
	// sarılır (kenarlık 2 + iç boşluk 6 sütun).
	icW := 72
	if m.w > 0 {
		icW = max(20, min(m.w-8, icW))
	}
	for i, s := range satirlar {
		s = ui.Wrap(s, icW)
		if i == 0 {
			b.WriteString(s + "\n")
		} else {
			b.WriteString(ui.S.Muted.Render(s) + "\n")
		}
	}
	b.WriteString("\n")
	if m.pending {
		b.WriteString(ui.S.Accent.Render(m.spinner.View()) + ui.S.Muted.Render(dil.T(" İşleniyor…")) + "\n\n")
	}
	b.WriteString(ui.S.Key.Render("y") + " " + ui.S.KeyDesc.Render(dil.T("onayla")) + ui.S.Subtle.Render(" · ") + ui.S.Key.Render("n") + " " + ui.S.KeyDesc.Render(dil.T("vazgeç")))
	return m.diyalog(b.String(), ui.P.Error)
}

// viewYardim, ? ile açılan kısayol kutusudur; içeriği önceki ekranınkidir.
func (m model) viewYardim() string {
	var b strings.Builder
	b.WriteString(ui.S.Title.Render(dil.T("Kısayollar")) + "\n\n")
	for _, k := range m.kisayollarIcin(m.yardimGeriEkran) {
		b.WriteString(ui.S.Key.Render(fmt.Sprintf("%-16s", k.tus)) + " " + k.aciklama + "\n")
	}
	b.WriteString("\n" + ui.S.Muted.Render(dil.T("Her zaman: ctrl+c çıkış")) + "\n\n")
	b.WriteString(ui.S.Subtle.Render(dil.T("herhangi bir tuş — geri dön")))
	return m.diyalog(b.String(), ui.P.Accent)
}
