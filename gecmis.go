package main

// gecmis.go — grup mesajlarını kaçırmamak.
//
// Grup protokolü özel mesajlarla (DM) taşınır. DeltaX önceden DM'leri
// yalnızca canlı dinliyordu (PollDM): kapalıyken gelen bir davet, üye
// listesi güncellemesi, ayrılma ya da grup kapatma bildirimi bir daha hiç
// görülmüyordu; açıkken bile aynı anda gelen iki mesajdan biri
// kaybolabiliyordu. AND artık DM geçmişini saklıyor: DeltaX geçmişi okur,
// nereye kadar işlediğini diske yazar ve bir sonraki açılışta oradan devam
// eder. Geçmişi olmayan eski bir AND'da canlı dinlemeye düşer.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lucian95511/and/internal/dosya"
	"github.com/lucian95511/and/internal/pluginapi"
)

const (
	// pullGecerlilik: bundan eski bir "çek" isteği yanıtlanmaz. Gönderim
	// isteyenin çevrimiçi olmasını gerektirir; gerekirse yeniden ister.
	pullGecerlilik = time.Hour
	// davetGecerlilik: bir davetin kabulü gönderildikten sonra bu süre
	// içinde gelmeli. Giden kutusu bir daveti 7 güne kadar bekletebilir;
	// davet edilenin karar vermesi de zaman alabilir.
	davetGecerlilik = 30 * 24 * time.Hour
	// gecmisAraligi: açıkken geçmişin yeniden okunma aralığı.
	gecmisAraligi = 2 * time.Second
)

type dmGecmisMsg struct {
	kayitlar []pluginapi.DMRecord
	err      error
	// istendi, geçmişin istendiği an: bundan önce gelen her mesaj yanıttadır.
	istendi time.Time
}

// gecmisCek, geçmişi imlecin kaydından (sonId) itibaren ister: AND her turda
// tüm geçmişi göndermez. Kayıt artık geçmişte yoksa ya da AND bu süzmeyi
// bilmiyorsa tamamı gelir; dmImleci.yeniler ikisini de doğru işler.
func gecmisCek(c istemci, sonID string) dmGecmisMsg {
	istendi := time.Now().UTC()
	k, err := c.DMHistoryFrom(sonID)
	return dmGecmisMsg{kayitlar: k, err: err, istendi: istendi}
}

func gecmisCekCmd(c istemci, sonID string) tea.Cmd {
	return func() tea.Msg { return gecmisCek(c, sonID) }
}

func gecmisCekCmdAfter(c istemci, sonID string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(gecmisAraligi)
		return gecmisCek(c, sonID)
	}
}

// dmGecmisiGeldi, AND'ın DM geçmişinde imleçten sonra gelen grup
// mesajlarını sırayla işler. İlk çalıştırmada (imleç yokken) önceki mesajlar
// işlenmez: eski sürüm onları canlı olarak zaten işlemişti; yeniden işlemek
// eski bir üye listesini daha yeni bir durumun üzerine yazabilir ya da eski
// bir "çek" isteğini yeniden çalıştırabilirdi.
func (m model) dmGecmisiGeldi(msg dmGecmisMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if pluginapi.IsNotFound(msg.err) {
			return m, dmPollCmd(m.client) // geçmişi olmayan eski AND: yalnızca canlı DM'ler
		}
		return m, gecmisCekCmdAfter(m.client, m.imlec.ID)
	}
	if m.imlec.bos() {
		m.imlec = dmImleci{Zaman: msg.istendi}.ilerle(msg.kayitlar)
		m.imlec.kaydet(m.dataDir)
		return m, gecmisCekCmdAfter(m.client, m.imlec.ID)
	}
	yeniler := m.imlec.yeniler(msg.kayitlar)
	var cmds []tea.Cmd
	for _, r := range yeniler {
		if r.Outgoing {
			continue
		}
		z, ok := zarfCoz(r.Text)
		if !ok {
			continue
		}
		var cmd tea.Cmd
		m, cmd = m.zarfIsle(z, r.PeerID, r.At)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(yeniler) > 0 {
		m.imlec = m.imlec.ilerle(yeniler)
		m.imlec.kaydet(m.dataDir)
	}
	return m, tea.Batch(append(cmds, gecmisCekCmdAfter(m.client, m.imlec.ID))...)
}

// ─── İmleç ────────────────────────────────────────────────────────────────────

// dmImleci, DM geçmişinde nereye kadar işlendiğidir: en son görülen kaydın
// kimliği ve zamanı. Sıfır değeri "hiç çalışmadı" demektir.
type dmImleci struct {
	ID    string    `json:"id,omitempty"`
	Zaman time.Time `json:"zaman"`
}

func (im dmImleci) bos() bool { return im.ID == "" && im.Zaman.IsZero() }

// yeniler, kayitlar içinde imleçten sonra gelenlerdir. Geçmiş geliş
// sırasıyla tutulur: imlecin kaydı hâlâ oradaysa ondan sonrakiler yenidir
// (saat geri alınsa bile). Kayıt geçmişten düşmüşse ya da AND kimlik
// göndermiyorsa imlecin zamanından sonra gelen ilk kayıttan itibaren
// alınır.
func (im dmImleci) yeniler(kayitlar []pluginapi.DMRecord) []pluginapi.DMRecord {
	if im.ID != "" {
		for i := len(kayitlar) - 1; i >= 0; i-- {
			if kayitlar[i].ID == im.ID {
				return kayitlar[i+1:]
			}
		}
	}
	for i, r := range kayitlar {
		if r.At.After(im.Zaman) {
			return kayitlar[i:]
		}
	}
	return nil
}

// ilerle, imleci kayitlar'ın sonuncusuna taşır; kayitlar boşsa imleç
// olduğu gibi kalır.
func (im dmImleci) ilerle(kayitlar []pluginapi.DMRecord) dmImleci {
	if n := len(kayitlar); n > 0 {
		return dmImleci{ID: kayitlar[n-1].ID, Zaman: kayitlar[n-1].At}
	}
	return im
}

func imlecYolu(dataDir string) string {
	return filepath.Join(kokDizin(dataDir), "dm_imleci.json")
}

func imleciOku(dataDir string) dmImleci {
	var im dmImleci
	if data, err := os.ReadFile(imlecYolu(dataDir)); err == nil {
		_ = json.Unmarshal(data, &im)
	}
	return im
}

func (im dmImleci) kaydet(dataDir string) {
	jsonYaz(dataDir, imlecYolu(dataDir), im)
}

// ─── Gönderilen davetler ──────────────────────────────────────────────────────

// davetDefteri, hangi grup için hangi eşe ne zaman davet gönderildiğidir
// (grupID → peerID → zaman). Bir "kabul" yanıtı yalnızca gerçekten davet
// edilmiş bir eşten ve davetGecerlilik içinde gelirse uygulanır; aksi halde
// hiç davet edilmemiş biri bile "kabul ettim" diyerek gruba katılabilirdi.
// Diskte durur: DeltaX kapalıyken gelen meşru bir kabul, yeniden açılınca
// da tanınır.
type davetDefteri map[string]map[string]time.Time

func davetDefteriYolu(dataDir string) string {
	return filepath.Join(kokDizin(dataDir), "gonderilen_davetler.json")
}

// davetDefteriniOku, defteri yükler; süresi dolan kayıtlar düşer.
func davetDefteriniOku(dataDir string, simdi time.Time) davetDefteri {
	d := davetDefteri{}
	if data, err := os.ReadFile(davetDefteriYolu(dataDir)); err == nil {
		_ = json.Unmarshal(data, &d)
	}
	for g, kisiler := range d {
		for p, t := range kisiler {
			if simdi.Sub(t) > davetGecerlilik {
				delete(kisiler, p)
			}
		}
		if len(kisiler) == 0 {
			delete(d, g)
		}
	}
	return d
}

func (d davetDefteri) kaydet(dataDir string) {
	jsonYaz(dataDir, davetDefteriYolu(dataDir), d)
}

func (d *davetDefteri) ekle(grupID, peerID string, t time.Time) {
	if *d == nil {
		*d = davetDefteri{}
	}
	if (*d)[grupID] == nil {
		(*d)[grupID] = map[string]time.Time{}
	}
	(*d)[grupID][peerID] = t
}

// gecerli, alindi anında gelen bir yanıt için peerID'ye grupID için
// davetGecerlilik içinde davet gönderilmiş olup olmadığıdır.
func (d davetDefteri) gecerli(grupID, peerID string, alindi time.Time) bool {
	t, ok := d[grupID][peerID]
	return ok && alindi.Sub(t) <= davetGecerlilik
}

func (d davetDefteri) sil(grupID, peerID string) {
	if d[grupID] != nil {
		delete(d[grupID], peerID)
		if len(d[grupID]) == 0 {
			delete(d, grupID)
		}
	}
}

// jsonYaz, v'yi yalnızca sahibinin okuyabileceği bir dosyaya yazar. Hata
// sessizce yutulur: bu dosyalar kaybolursa en kötü ihtimalle bir mesaj
// yeniden işlenmez ya da bir davetin yeniden gönderilmesi gerekir.
func jsonYaz(dataDir, yol string, v any) {
	if dataDir == "" {
		return
	}
	if err := os.MkdirAll(kokDizin(dataDir), 0o700); err != nil {
		return
	}
	if data, err := json.Marshal(v); err == nil {
		_ = dosya.AtomikYaz(yol, data, 0o600)
	}
}
