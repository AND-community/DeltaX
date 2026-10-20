// AND eklentisi: DeltaX — gruplar kurup kendi sürüm kontrol sistemiyle
// birlikte proje geliştirmeyi sağlar. Tüm senkronizasyon, AND'ın merkezi
// forum/sohbet kanallarına hiç değmeden, yalnızca grup üyeleri arasında
// doğrudan (DM + dosya aktarımı) ve grup sırrıyla uçtan uca şifreli olarak
// gerçekleşir — grubun "kendi özel ağı" budur.
//
// Derleme:
//
//	go build -o and-plugin-DeltaX[.exe] ./Eklentiler/DeltaX
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lucian95511/and/internal/pluginapi"
	"github.com/lucian95511/and/internal/pluginui"
	"github.com/lucian95511/and/internal/ui"
)

// describeErr, ham bir Go/HTTP hatasını kullanıcıya gösterilecek anlaşılır bir
// metne çevirir. Tanınmayan hata sınıfları için (icat etmek yerine) ham
// err.Error() metnine düşülür — bilgi kaybetmemek daha önemli.
func describeErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// internal/pluginapi.Client'ın sunucu tarafı hataları "api: <AND'ın kendi
	// Türkçe mesajı>" biçiminde sarmalanır (bkz. internal/pluginapi/api.go)
	// — önek zaten bilgi taşımıyor, atılabilir.
	if rest, ok := strings.CutPrefix(msg, "api: "); ok {
		return rest
	}

	// Bağlantı düzeyi hatalar (AND kapalı/AND_ENABLE_EXTERNAL_API kapalı/pipe
	// koptu) Go'nun net/http'sinden ham, İngilizce/işletim sistemi diliyle
	// gelir (ör. "dial unix ...: connectex: ..."); ortak alt dizgelerle tespit
	// edip tek, anlaşılır bir mesaja indirgiyoruz.
	lower := strings.ToLower(msg)
	for _, izVar := range []string{"connect", "dial", "refused", "timeout", "eof", "no such host", "broken pipe"} {
		if strings.Contains(lower, izVar) {
			return "AND'a bağlanılamadı — bağlantı kopmuş olabilir."
		}
	}

	if errors.Is(err, fs.ErrNotExist) {
		return "Dosya/klasör bulunamadı."
	}
	if errors.Is(err, fs.ErrPermission) {
		return "İzin reddedildi (dosya başka bir uygulama tarafından kullanılıyor olabilir)."
	}

	return msg
}

var manifest = pluginapi.Manifest{
	Name:        "DeltaX",
	Label:       "DeltaX",
	Version:     "1.4.1",
	Description: "Özel gruplar kurup kendi sürüm kontrol sistemiyle birlikte proje geliştirme — merkezi ağ dışında, yalnızca grup üyeleri arasında uçtan uca şifreli senkronizasyon",
	Author:      "AND",
	Permissions: []string{pluginapi.IzinDM, pluginapi.IzinDosya},
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--manifest" {
		data, _ := json.Marshal(manifest)
		fmt.Println(string(data))
		return
	}

	client, err := pluginapi.NewClientFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeltaX:", err)
		os.Exit(1)
	}

	identity, err := client.Identity()
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeltaX: kimlik alınamadı:", err)
		os.Exit(1)
	}

	dataDir := pluginapi.DataDir()
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "DeltaX: AND_DATA_DIR tanımlı değil")
		os.Exit(1)
	}

	// AND'ın o an aktif olan temasını (ör. Pembiş) çek ve kendi stillerimize
	// uygula — böylece eklenti, ana uygulamayla aynı renk şemasında açılır.
	// Eski bir AND sürümünde bu uç nokta yoksa sessizce varsayılan renklerde
	// kalınır.
	if tema, err := client.GetTheme(); err == nil {
		uygulaTema(tema)
	}

	m := newModel(client, identity, dataDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "DeltaX:", err)
		os.Exit(1)
	}
}

// ─── Terminal güvenliği ───────────────────────────────────────────────────────
//
// Grup üyelerinden gelen her string (grup adı, kullanıcı adı, commit
// mesajı, dosya yolu vb.) güvenilmezdir; terminale yazmadan önce ANSI
// escape dizileri temizlenir — bkz. internal/pluginui.Safe.

func safe(s string) string { return pluginui.Safe(s) }

// klasoruAc, verilen dizini işletim sistemine özgü dosya yöneticisinde açar.
func klasoruAc(yol string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", yol).Start()
	case "darwin":
		return exec.Command("open", yol).Start()
	default:
		return exec.Command("xdg-open", yol).Start()
	}
}

// ─── Stiller ──────────────────────────────────────────────────────────────────

// Stiller AND'ın ortak paletinden (internal/ui) türetilir — DeltaX, AND'ın
// TUI'si ve diğer eklentilerle aynı temada görünür.
var (
	stTitle   lipgloss.Style
	stBold    lipgloss.Style
	stSel     lipgloss.Style
	stNormal  lipgloss.Style
	stOk      lipgloss.Style
	stErr     lipgloss.Style
	stDim     lipgloss.Style
	stMuted   lipgloss.Style
	stFocused lipgloss.Style
	stBorder  lipgloss.Style
	stAdd     lipgloss.Style
	stDel     lipgloss.Style
	stMod     lipgloss.Style
	stWarn    lipgloss.Style
)

func init() { stilleriKur() }

func stilleriKur() {
	stTitle = ui.S.Title
	stBold = ui.S.Bold
	stSel = ui.S.Selected
	stNormal = lipgloss.NewStyle()
	stOk = ui.S.Success
	stErr = ui.S.Error
	stDim = ui.S.Muted
	stMuted = ui.S.Muted
	stFocused = ui.S.Title
	stBorder = ui.S.CardFocus
	stAdd = ui.S.Success
	stDel = ui.S.Error
	stMod = ui.S.Warning
	stWarn = ui.S.Warning.Bold(true)
}

// uygulaTema, AND'ın GET /api/v1/theme'den döndürdüğü temayı ortak palete
// uygular ve stilleri yeniden kurar. Hata/başarı/değişiklik (Ok/Err/Warn/
// Add/Del/Mod) renkleri temadan etkilenmez — ui.Apply'ın kuralı.
func uygulaTema(t pluginapi.ThemeReq) {
	ui.Apply(ui.Theme(t))
	stilleriKur()
}
