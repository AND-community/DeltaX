package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/lucian95511/and/protokol/altin"
)

// DeltaX'in grup protokolünü iki gerçekleme konuşur: bu eklenti (Go) ve AND'ın
// VS Code eklentisi (TypeScript, and-vscode/src/live/deltax). İkisi de
// protokol/testdata/deltax'teki ortak girdilerden (grup sırrı, açık metin)
// örnek üretir ve ötekinin ürettiğini çözer: Go'nun şifrelediği paketi
// TypeScript, TypeScript'in kodladığı zarfı Go açabilmelidir.
//
// Örnekleri yeniden üretmek için:
//   AND_ALTIN_GUNCELLE=1 go test ./Eklentiler/DeltaX -run Altin
//   (cd and-vscode && AND_ALTIN_GUNCELLE=1 npm test)
//   go test ./Eklentiler/DeltaX -run Altin

func altinDavet(sir string) davetPayload {
	return davetPayload{
		GrupAd: "Altın <grup> & ekip",
		Sir:    sir,
		Kurucu: Uye{Ad: "kurucu", PeerID: "12D3KooWKurucuAltin"},
		Uyeler: []Uye{
			{Ad: "kurucu", PeerID: "12D3KooWKurucuAltin"},
			{Ad: "üye", PeerID: "12D3KooWUyeAltin", PubKey: "abcd"},
		},
		Gonderen: Uye{Ad: "kurucu", PeerID: "12D3KooWKurucuAltin"},
	}
}

func davetDogrula(t *testing.T, kaynak string, z kontrolZarfi, sir string) {
	t.Helper()
	if z.Tip != mDavet || z.GrupID != "grup-altin" {
		t.Fatalf("%s zarfı beklenmedik: %+v", kaynak, z)
	}
	var p davetPayload
	if err := json.Unmarshal(z.Payload, &p); err != nil {
		t.Fatalf("%s davet yükü çözülemedi: %v", kaynak, err)
	}
	want := altinDavet(sir)
	if p.GrupAd != want.GrupAd || p.Sir != sir || p.Kurucu != want.Kurucu || len(p.Uyeler) != 2 || p.Uyeler[1] != want.Uyeler[1] || p.Gonderen != want.Gonderen {
		t.Fatalf("%s davet yükü beklenmedik: %+v", kaynak, p)
	}
}

func TestAltinDeltaX(t *testing.T) {
	sir := strings.TrimSpace(string(altin.Oku(t, "deltax/sir.txt")))
	acik := altin.Oku(t, "deltax/acik.txt")
	guncelle := os.Getenv(altin.GuncelleDegiskeni) == "1"

	yuk, _ := json.Marshal(altinDavet(sir))
	zarf, err := zarfKodla(kontrolZarfi{Tip: mDavet, GrupID: "grup-altin", Payload: yuk})
	if err != nil {
		t.Fatal(err)
	}
	altin.Karsilastir(t, "deltax/zarf_go.txt", []byte(zarf))

	// Şifreli örnek rastgele nonce taşır: yalnızca güncellemede üretilir,
	// her çalışmada çözülebildiği sınanır.
	if guncelle {
		blob, err := sirliSifrele(sir, acik)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(altin.Dosya("deltax/sifreli_go.bin"), blob, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if cozulen, err := sirliCoz(sir, altin.Oku(t, "deltax/sifreli_go.bin")); err != nil || !bytes.Equal(cozulen, acik) {
		t.Fatalf("Go kendi şifreli örneğini çözmeli: %v", err)
	}

	// TypeScript'in ürettikleri (güncellemenin ilk adımında henüz yoksa atlanır).
	if _, err := os.Stat(altin.Dosya("deltax/zarf_ts.txt")); guncelle && err != nil {
		t.Skip("TypeScript örnekleri henüz üretilmedi")
	}
	tsZarf, ok := zarfCoz(string(altin.Oku(t, "deltax/zarf_ts.txt")))
	if !ok {
		t.Fatal("Go, TypeScript'in kodladığı zarfı çözebilmeli")
	}
	davetDogrula(t, "TypeScript", tsZarf, sir)
	if cozulen, err := sirliCoz(sir, altin.Oku(t, "deltax/sifreli_ts.bin")); err != nil || !bytes.Equal(cozulen, acik) {
		t.Fatalf("Go, TypeScript'in şifrelediği paketi çözebilmeli: %v", err)
	}
	kurcalanmis := append([]byte(nil), altin.Oku(t, "deltax/sifreli_ts.bin")...)
	kurcalanmis[len(kurcalanmis)-1] ^= 1
	if _, err := sirliCoz(sir, kurcalanmis); err == nil {
		t.Fatal("değiştirilmiş şifreli veri reddedilmeli")
	}
}
