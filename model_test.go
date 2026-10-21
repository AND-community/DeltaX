package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/lucian95511/and/internal/pluginapi"
)

// TestDalDegistirCmdBozukHedefCommitCalismaDizininiSilmez, headTreeOf'un
// artık commit okuma hatasını "boş ağaç" ile karıştırmadığını doğrudan
// dalDegistirCmd üzerinden doğrular: hedef daldaki commit nesnesi diskten
// silinmiş/bozuksa, dal değiştirme bir hata döndürmeli ve — eski,
// hatanın yutulduğu davranışın aksine — çalışma dizinini SESSİZCE
// boşaltmamalıdır.
func TestDalDegistirCmdBozukHedefCommitCalismaDizininiSilmez(t *testing.T) {
	dataDir, projeID, calismaDir := yeniTestProjesi(t)
	yazDosya(t, calismaDir, "a.txt", "içerik")

	treeHash, _, _, err := agacOlustur(dataDir, projeID, calismaDir)
	if err != nil {
		t.Fatal(err)
	}
	mevcutHead, err := commitOlustur(dataDir, projeID, treeHash, nil, "test", "ilk kayıt")
	if err != nil {
		t.Fatal(err)
	}

	// Hedef dal, aslında var olmayan/bozuk bir commit hash'ine işaret
	// ediyor (ör. senkronize edilmemiş bir uzak dal referansı).
	sahteHedefHash := "0000000000000000000000000000000000000000000000000000000000000000"[:64]

	cmd := dalDegistirCmd(dataDir, projeID, calismaDir, mevcutHead, "uzak-dal", sahteHedefHash)
	msg := cmd()
	sonuc, ok := msg.(dalDegistirSonucu)
	if !ok {
		t.Fatalf("beklenmeyen mesaj tipi: %T", msg)
	}
	if sonuc.err == nil {
		t.Fatal("bozuk hedef commit için bir hata döndürülmeliydi")
	}

	data, err := os.ReadFile(calismaDir + "/a.txt")
	if err != nil {
		t.Fatalf("çalışma dizini sessizce silinmiş: a.txt okunamadı: %v", err)
	}
	if string(data) != "içerik" {
		t.Fatalf("a.txt içeriği değişmemeliydi, geldi: %q", data)
	}
}

func TestZarfIsleSahteKurucudanRosterGuncellemesiniReddeder(t *testing.T) {
	dataDir := t.TempDir()

	gercek := Uye{Ad: "gercekKurucu", PeerID: "peer-gercek"}
	g := Grup{ID: "grup1", Ad: "Orijinal Ad", Sir: mustSir(t), KurucuAd: gercek.Ad, KurucuPeer: gercek.PeerID, Uyeler: []Uye{gercek}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	// Grubun gerçek kurucusu olmayan biri, sahte bir Kurucu.PeerID ile grup
	// adını ve üye listesini değiştirmeye çalışıyor. Payload'ın iddia ettiği
	// kimlik ile DM'in gerçekten geldiği doğrulanmış kimlik aynı (saldırgan
	// kendi gerçek kimliğiyle konuşuyor) — reddin nedeni "yetkili değil",
	// taklit değil.
	saldirgan := Uye{Ad: "saldirgan", PeerID: "peer-saldirgan"}
	payload, _ := json.Marshal(davetPayload{
		GrupAd:   "Ele Geçirilmiş Ad",
		Sir:      g.Sir,
		Kurucu:   Uye{Ad: "sahte", PeerID: "peer-sahte"}, // gerçek kurucuyla eşleşmiyor
		Gonderen: Uye{Ad: "sahte", PeerID: "peer-sahte"}, // ne kurucu ne yönetici
		Uyeler:   []Uye{saldirgan},
	})

	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-kurban"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mUyeGuncelle, GrupID: g.ID, Payload: payload}, "peer-sahte", time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.Ad != "Orijinal Ad" {
		t.Fatalf("grup adı sahte kurucudan gelen mesajla değişmemeliydi, geldi: %q", sonra.Ad)
	}
	if len(sonra.Uyeler) != 1 || sonra.Uyeler[0].PeerID != gercek.PeerID {
		t.Fatalf("üye listesi sahte mesajla değişmemeliydi: %+v", sonra.Uyeler)
	}
}

func TestZarfIsleGercekKurucudanRosterGuncellemesiniKabulEder(t *testing.T) {
	dataDir := t.TempDir()

	gercek := Uye{Ad: "gercekKurucu", PeerID: "peer-gercek"}
	g := Grup{ID: "grup1", Ad: "Orijinal Ad", Sir: mustSir(t), KurucuAd: gercek.Ad, KurucuPeer: gercek.PeerID, Uyeler: []Uye{gercek}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	yeniUye := Uye{Ad: "yeniUye", PeerID: "peer-yeni"}
	payload, _ := json.Marshal(davetPayload{
		GrupAd:   "Güncellenmiş Ad",
		Sir:      g.Sir,
		Kurucu:   gercek,
		Gonderen: gercek, // gerçek kurucu gönderiyor
		Uyeler:   []Uye{gercek, yeniUye},
	})

	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-kurban"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mUyeGuncelle, GrupID: g.ID, Payload: payload}, gercek.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.Ad != "Güncellenmiş Ad" {
		t.Fatalf("grup adı gerçek kurucudan gelen mesajla güncellenmeliydi, geldi: %q", sonra.Ad)
	}
	if len(sonra.Uyeler) != 2 {
		t.Fatalf("üye listesi 2 üye içermeliydi: %+v", sonra.Uyeler)
	}
}

// TestZarfIsleRosterGuncellemesiTaklitEdilenKurucudanReddeder,
// zarfIsle'ın verifiedSender kontrolünü doğrudan test eder: saldırgan kendi
// gerçek (doğrulanmış) kimliğiyle DM gönderiyor ama payload içine GERÇEK
// kurucunun PeerID'sini yazıyor. Yalnızca g.yetkiliMi(dp.Gonderen.PeerID)
// kontrolüyle bu payload "yetkiliden geliyormuş" gibi kabul edilirdi —
// verifiedSender karşılaştırması olmadan bu, herhangi bir peer'ın kurucu
// kimliğine bürünüp rostere el koymasına izin verirdi.
func TestZarfIsleRosterGuncellemesiTaklitEdilenKurucudanReddeder(t *testing.T) {
	dataDir := t.TempDir()

	gercek := Uye{Ad: "gercekKurucu", PeerID: "peer-gercek"}
	g := Grup{ID: "grup1", Ad: "Orijinal Ad", Sir: mustSir(t), KurucuAd: gercek.Ad, KurucuPeer: gercek.PeerID, Uyeler: []Uye{gercek}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	saldirgan := Uye{Ad: "saldirgan", PeerID: "peer-saldirgan"}
	payload, _ := json.Marshal(davetPayload{
		GrupAd:   "Ele Geçirilmiş Ad",
		Sir:      g.Sir,
		Kurucu:   gercek,
		Gonderen: gercek, // payload GERÇEK kurucu gönderiyormuş gibi iddia ediyor
		Uyeler:   []Uye{saldirgan},
	})

	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-kurban"}}
	// DM aslında saldırganın kendi doğrulanmış kimliğinden geldi — kurucudan değil.
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mUyeGuncelle, GrupID: g.ID, Payload: payload}, saldirgan.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.Ad != "Orijinal Ad" {
		t.Fatalf("grup adı taklit edilen kurucu mesajıyla değişmemeliydi, geldi: %q", sonra.Ad)
	}
	if len(sonra.Uyeler) != 1 || sonra.Uyeler[0].PeerID != gercek.PeerID {
		t.Fatalf("üye listesi taklit mesajıyla değişmemeliydi: %+v", sonra.Uyeler)
	}
}

func TestZarfIsleAyrilKurucuUyeyiRosterdenCikarir(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	ayrilan := Uye{Ad: "ayrilan", PeerID: "peer-ayrilan"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu, ayrilan}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(ayrilPayload{Uye: ayrilan})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: kurucu.PeerID}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mAyril, GrupID: g.ID, Payload: payload}, ayrilan.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.uyeMi(ayrilan.PeerID) {
		t.Fatalf("ayrilan üye hâlâ roster'da: %+v", sonra.Uyeler)
	}
	if !sonra.uyeMi(kurucu.PeerID) {
		t.Fatal("kurucu yanlışlıkla roster'dan çıkarılmış")
	}
}

// TestZarfIsleAyrilBaskasiAdinaTaklitEdilerekKabulEtmez, mAyril'in artık
// yalnızca gerçekten ayrılan üyenin kendisinden (verifiedSender ==
// ap.Uye.PeerID) kabul edildiğini doğrular — eskiden yalnızca ALICI'nın
// (m.identity.PeerID) yetkili olup olmadığına bakılıyordu, GÖNDEREN hiç
// doğrulanmıyordu; bu da herhangi bir peer'ın "X ayrıldı" diyerek rastgele
// bir üyeyi attırabilmesine izin veriyordu.
func TestZarfIsleAyrilBaskasiAdinaTaklitEdilerekKabulEtmez(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	kurban := Uye{Ad: "kurban", PeerID: "peer-kurban-uye"}
	saldirgan := Uye{Ad: "saldirgan", PeerID: "peer-saldirgan"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu, kurban, saldirgan}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	// Saldırgan kendi (doğrulanmış) kimliğinden, ama payload'da kurban'ın
	// ayrıldığını iddia eden bir mesaj gönderiyor.
	payload, _ := json.Marshal(ayrilPayload{Uye: kurban})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: kurucu.PeerID}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mAyril, GrupID: g.ID, Payload: payload}, saldirgan.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sonra.uyeMi(kurban.PeerID) {
		t.Fatal("kurban, kendisi göndermediği bir 'ayrıldı' mesajıyla roster'dan çıkarılmış")
	}
}

func TestZarfIsleGrupKapatSahteKurucudanKabulEtmez(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(grupKapatPayload{Gonderen: Uye{PeerID: "peer-sahte"}})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-uye"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mGrupKapat, GrupID: g.ID, Payload: payload}, "peer-sahte", time.Now())
	_ = m

	if _, err := grupOku(dataDir, g.ID); err != nil {
		t.Fatalf("sahte kurucudan gelen kapatma isteğiyle grup silinmemeliydi: %v", err)
	}
}

// TestZarfIsleGrupKapatTaklitEdilenKurucudanKabulEtmez, mGrupKapat için de
// aynı taklit senaryosunu (payload gerçek kurucunun kimliğini iddia ediyor,
// ama DM'in doğrulanmış gönderenin başka biri) test eder.
func TestZarfIsleGrupKapatTaklitEdilenKurucudanKabulEtmez(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(grupKapatPayload{Gonderen: kurucu}) // payload gerçek kurucu iddia ediyor
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-uye"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mGrupKapat, GrupID: g.ID, Payload: payload}, "peer-saldirgan", time.Now()) // ama gerçek gönderen saldırgan
	_ = m

	if _, err := grupOku(dataDir, g.ID); err != nil {
		t.Fatalf("taklit edilen kurucudan gelen kapatma isteğiyle grup silinmemeliydi: %v", err)
	}
}

func TestZarfIsleGrupKapatGercekKurucudanGrubuSiler(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(grupKapatPayload{Gonderen: kurucu})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-uye"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mGrupKapat, GrupID: g.ID, Payload: payload}, kurucu.PeerID, time.Now())
	_ = m

	if _, err := grupOku(dataDir, g.ID); err == nil {
		t.Fatal("gerçek kurucudan gelen kapatma isteğiyle grup silinmeliydi")
	}
}

func TestZarfIsleGrupKapatYoneticidenDeKabulEder(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	yonetici := Uye{Ad: "yonetici", PeerID: "peer-yonetici"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu, yonetici}, Yoneticiler: []Uye{yonetici}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	// Kurucu ulaşılamaz durumda olsa bile, kurucunun daha önce atadığı
	// bir yönetici grubu kapatabilmeli (yedek kurucu senaryosu).
	payload, _ := json.Marshal(grupKapatPayload{Gonderen: yonetici})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: "peer-uye"}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mGrupKapat, GrupID: g.ID, Payload: payload}, yonetici.PeerID, time.Now())
	_ = m

	if _, err := grupOku(dataDir, g.ID); err == nil {
		t.Fatal("yöneticiden gelen kapatma isteğiyle grup silinmeliydi")
	}
}

// TestZarfIsleDavetYaniti* — mDavetYanit artık yalnızca (a) kabul mesajının
// gerçekten kabul eden üyeden geldiğini (verifiedSender == yp.Uye.PeerID)
// VE (b) bu admin'in bu peer'a bu grup için gerçekten bir davet göndermiş
// olduğunu (m.gonderilenDavetler) doğruladıktan sonra uyguluyor. Eskiden
// tek şart alıcının admin olmasıydı — hiç davet edilmemiş biri bile "kabul
// ediyorum" diyerek özel gruba kendini ekleyebilirdi.

func TestZarfIsleDavetYanitiDavetEdilmemisPeerKabulEtmez(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	davetsiz := Uye{Ad: "davetsiz", PeerID: "peer-davetsiz"}
	payload, _ := json.Marshal(davetYanitPayload{Kabul: true, Uye: davetsiz})
	m := model{dataDir: dataDir, identity: pluginapi.IdentityInfo{PeerID: kurucu.PeerID}, gonderilenDavetler: davetDefteri{}}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mDavetYanit, GrupID: g.ID, Payload: payload}, davetsiz.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.uyeMi(davetsiz.PeerID) {
		t.Fatal("hiç davet edilmemiş bir peer, sahte bir 'kabul ediyorum' mesajıyla gruba eklenmiş")
	}
}

func TestZarfIsleDavetYanitiGercekDaveteKabulEder(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	davetli := Uye{Ad: "davetli", PeerID: "peer-davetli"}
	payload, _ := json.Marshal(davetYanitPayload{Kabul: true, Uye: davetli})
	m := model{
		dataDir:            dataDir,
		identity:           pluginapi.IdentityInfo{PeerID: kurucu.PeerID},
		gonderilenDavetler: davetDefteri{g.ID: {davetli.PeerID: time.Now()}},
	}
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mDavetYanit, GrupID: g.ID, Payload: payload}, davetli.PeerID, time.Now())

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !sonra.uyeMi(davetli.PeerID) {
		t.Fatal("gerçekten davet edilmiş bir peer'ın kabulü uygulanmadı")
	}
	if _, ok := m.gonderilenDavetler[g.ID][davetli.PeerID]; ok {
		t.Fatal("kullanılan davet kaydı tüketilmeliydi (delete edilmeliydi)")
	}
}

// TestZarfIsleDavetYanitiTaklitEdilerekKabulEtmez, davet edilen kişinin
// PeerID'siyle payload'ı dolduran ama başka bir peer'dan gönderen bir
// saldırganın (davet gerçekten var olsa bile) kabul edilmediğini doğrular.
func TestZarfIsleDavetYanitiTaklitEdilerekKabulEtmez(t *testing.T) {
	dataDir := t.TempDir()

	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	g := Grup{ID: "grup1", Ad: "Grup", Sir: mustSir(t), KurucuAd: kurucu.Ad, KurucuPeer: kurucu.PeerID, Uyeler: []Uye{kurucu}}
	if err := grupKaydet(dataDir, g); err != nil {
		t.Fatal(err)
	}

	davetli := Uye{Ad: "davetli", PeerID: "peer-davetli"}
	saldirgan := Uye{Ad: "saldirgan", PeerID: "peer-saldirgan"}
	payload, _ := json.Marshal(davetYanitPayload{Kabul: true, Uye: davetli}) // payload davetli iddia ediyor
	m := model{
		dataDir:            dataDir,
		identity:           pluginapi.IdentityInfo{PeerID: kurucu.PeerID},
		gonderilenDavetler: davetDefteri{g.ID: {davetli.PeerID: time.Now()}},
	}
	// ama DM gerçekte saldırgandan geldi
	m, _ = m.zarfIsle(kontrolZarfi{Tip: mDavetYanit, GrupID: g.ID, Payload: payload}, saldirgan.PeerID, time.Now())
	_ = m

	sonra, err := grupOku(dataDir, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonra.uyeMi(davetli.PeerID) {
		t.Fatal("taklit edilen kabul mesajıyla davetli üye eklenmemeliydi")
	}
}

// TestCommitSonucuBaskaProjeyeGecildiyseEskiProjeyeYazmaz, pending-guard
// boşluğunu doğrudan test eder: proje A için bir commitCmd sonucu beklenirken
// (m.pending==true) kullanıcı proje B'ye geçer (gezinme tuşları pending'i
// kontrol etmez); A için gecikmiş sonuç geldiğinde artık B'nin dosyasını
// A'nın commit hash'iyle EZMEMELİDİR — projeSonucuGuncelMi kontrolü bunu
// sağlar.
func TestCommitSonucuBaskaProjeyeGecildiyseEskiProjeyeYazmaz(t *testing.T) {
	dataDir := t.TempDir()

	projeA := Proje{ID: rastgeleID(), GrupID: "g1", Ad: "A", Dallar: map[string]string{"ana": ""}, AktifDal: "ana"}
	projeB := Proje{ID: rastgeleID(), GrupID: "g1", Ad: "B", Dallar: map[string]string{"ana": "eskiHashB"}, AktifDal: "ana"}
	if err := projeKaydet(dataDir, projeA); err != nil {
		t.Fatal(err)
	}
	if err := projeKaydet(dataDir, projeB); err != nil {
		t.Fatal(err)
	}

	// Kullanıcı A projesindeyken bir kayıt (commit) işlemi başlatıyor
	// (m.pending=true), sonra sonucu beklemeden B projesine geçiyor.
	guncel := projeB
	m := model{dataDir: dataDir, pending: true, aktifProje: &guncel}

	// A için (artık aktif olmayan proje) gecikmiş commitSonucu geliyor.
	sonuc := commitSonucu{projeID: projeA.ID, hash: "yeniHashA"}
	newM, _ := m.Update(sonuc)
	m = newM.(model)

	if m.pending {
		t.Fatal("pending, gecikmiş sonuç geldiğinde temizlenmeliydi")
	}
	if m.aktifProje.Dallar["ana"] != "eskiHashB" {
		t.Fatalf("B projesinin dalı, A'ya ait gecikmiş bir sonuçla değişmemeliydi: %q", m.aktifProje.Dallar["ana"])
	}

	sonraB, err := projeOku(dataDir, projeB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonraB.Dallar["ana"] != "eskiHashB" {
		t.Fatalf("diske de A'nın hash'i yanlışlıkla B için yazılmış: %q", sonraB.Dallar["ana"])
	}

	sonraA, err := projeOku(dataDir, projeA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sonraA.Dallar["ana"] != "" {
		t.Fatalf("A projesi de yanlışlıkla güncellenmiş olmamalı (hiç diske yazılmadı): %q", sonraA.Dallar["ana"])
	}
}

func TestYetkiliMiKurucuVeYoneticiyiKapsar(t *testing.T) {
	kurucu := Uye{Ad: "kurucu", PeerID: "peer-kurucu"}
	yonetici := Uye{Ad: "yonetici", PeerID: "peer-yonetici"}
	digeri := Uye{Ad: "digeri", PeerID: "peer-digeri"}
	g := Grup{KurucuPeer: kurucu.PeerID, Yoneticiler: []Uye{yonetici}}

	if !g.yetkiliMi(kurucu.PeerID) {
		t.Error("kurucu yetkili sayılmalı")
	}
	if !g.yetkiliMi(yonetici.PeerID) {
		t.Error("yönetici yetkili sayılmalı")
	}
	if g.yetkiliMi(digeri.PeerID) {
		t.Error("sıradan üye yetkili sayılmamalı")
	}
}
