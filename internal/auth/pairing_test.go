package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func differentPairCode(code string) string {
	normalized := strings.ReplaceAll(code, "-", "")
	first := byte('A')
	if normalized[0] == 'A' {
		first = 'B'
	}
	return string(first) + normalized[1:4] + "-" + normalized[4:]
}

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	if !limiter.allow("ip", now) || !limiter.allow("ip", now) {
		t.Fatal("los dos primeros intentos deben permitirse")
	}
	if limiter.allow("ip", now) || !limiter.blocked("ip", now) {
		t.Fatal("el tercer intento debe quedar bloqueado")
	}
	if !limiter.allow("otra-ip", now) {
		t.Fatal("otra clave no debe compartir el límite")
	}
	limiter.reset("ip")
	if limiter.blocked("ip", now) || !limiter.allow("ip", now) {
		t.Fatal("reset debe liberar la clave")
	}
	now = now.Add(time.Minute + time.Second)
	if limiter.blocked("ip", now) || !limiter.allow("ip", now) {
		t.Fatal("la ventana caducada debe liberar la clave")
	}
}

func TestVinculacionFlujoCompleto(t *testing.T) {
	svc, db, clock := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	pairing, err := svc.CreatePairing(ctx, "Salón", ClientAndroidTV, "198.51.100.10")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	if len(pairing.PairID) != 32 || pairing.Secret == "" ||
		len(pairing.Code) != 9 || pairing.Code[4] != '-' {
		t.Fatalf("vinculación inesperada: %+v", pairing)
	}
	if got := pairing.ExpiresAt.Sub(clock.Now()); got != pairingTTL {
		t.Fatalf("caducidad de la vinculación = %v", got)
	}

	var secretHash, codeHash string
	if err := db.QueryRow(
		`SELECT secret_hash, code_hash FROM device_pairings WHERE pair_id = ?`,
		pairing.PairID,
	).Scan(&secretHash, &codeHash); err != nil {
		t.Fatalf("leer hashes: %v", err)
	}
	normalized := strings.ReplaceAll(pairing.Code, "-", "")
	if secretHash != hashToken(pairing.Secret) || secretHash == pairing.Secret {
		t.Fatal("el secreto debe guardarse solo como hash")
	}
	if codeHash != hashToken(normalized) || codeHash == normalized {
		t.Fatal("el código debe guardarse solo como hash")
	}

	preview, err := svc.ApprovePairing(ctx, userID, strings.ToLower(pairing.Code), false, "198.51.100.10")
	if err != nil {
		t.Fatalf("vista previa: %v", err)
	}
	if preview.DeviceName != "Salón" || preview.Client != ClientAndroidTV || preview.Approved {
		t.Fatalf("vista previa inesperada: %+v", preview)
	}
	var approvedAt sql.NullString
	if err := db.QueryRow(
		`SELECT approved_at FROM device_pairings WHERE pair_id = ?`, pairing.PairID,
	).Scan(&approvedAt); err != nil {
		t.Fatalf("leer approved_at: %v", err)
	}
	if approvedAt.Valid {
		t.Fatal("confirm=false no debe aprobar la vinculación")
	}

	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.10"); !errors.Is(err, ErrPairingPending) {
		t.Fatalf("token pendiente = %v", err)
	}

	approved, err := svc.ApprovePairing(ctx, userID, pairing.Code, true, "198.51.100.10")
	if err != nil {
		t.Fatalf("aprobar: %v", err)
	}
	if !approved.Approved || approved.DeviceName != "Salón" {
		t.Fatalf("aprobación inesperada: %+v", approved)
	}

	session, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.10")
	if err != nil {
		t.Fatalf("canjear: %v", err)
	}
	if session.Token == "" || session.SessionID == 0 || session.Username != "ana" || session.Role != "user" {
		t.Fatalf("sesión de vinculación inesperada: %+v", session)
	}
	if got := session.ExpiresAt.Sub(clock.Now()); got != NativeSessionTTL {
		t.Fatalf("TTL de la sesión = %v, se esperaba %v", got, NativeSessionTTL)
	}

	var deviceName, client string
	if err := db.QueryRow(
		`SELECT device_name, client FROM sessions WHERE id = ?`, session.SessionID,
	).Scan(&deviceName, &client); err != nil {
		t.Fatalf("leer sesión creada: %v", err)
	}
	if deviceName != "Salón" || client != ClientAndroidTV {
		t.Fatalf("sesión creada con dispositivo %q y cliente %q", deviceName, client)
	}

	user, err := svc.Authenticate(ctx, session.Token)
	if err != nil {
		t.Fatalf("autenticar sesión vinculada: %v", err)
	}
	sessions, err := svc.ListSessions(ctx, userID, user.SessionID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	found := false
	for _, item := range sessions {
		if item.ID == session.SessionID && item.Current && item.Client == ClientAndroidTV {
			found = true
		}
	}
	if !found {
		t.Fatalf("la sesión vinculada no aparece en el listado: %+v", sessions)
	}

	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.10"); !errors.Is(err, ErrPairingExpired) {
		t.Fatalf("segundo canje = %v, se esperaba ErrPairingExpired", err)
	}
}

func TestVinculacionCaducada(t *testing.T) {
	svc, db, clock := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.20")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	clock.Advance(pairingTTL + time.Second)

	if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, false, "198.51.100.20"); !errors.Is(err, ErrPairingNotFound) {
		t.Fatalf("aprobar caducada = %v", err)
	}
	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.20"); !errors.Is(err, ErrPairingExpired) {
		t.Fatalf("canjear caducada = %v", err)
	}
}

func TestVinculacionSecretEquivocadoYLimite(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.30")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	for attempt := 0; attempt < pairTokenLimit; attempt++ {
		if _, err := svc.RedeemPairing(ctx, pairing.PairID, "secreto-equivocado", "198.51.100.30"); !errors.Is(err, ErrPairingNotFound) {
			t.Fatalf("intento %d = %v", attempt+1, err)
		}
	}
	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.30"); !errors.Is(err, ErrPairingRateLimited) {
		t.Fatalf("tras los fallos = %v", err)
	}

	clock.Advance(pairFailureWindow + time.Second)
	fresh, err := svc.CreatePairing(ctx, "TV", ClientAndroidTV, "198.51.100.30")
	if err != nil {
		t.Fatalf("CreatePairing tras ventana: %v", err)
	}
	if _, err := svc.RedeemPairing(ctx, fresh.PairID, fresh.Secret, "198.51.100.30"); !errors.Is(err, ErrPairingPending) {
		t.Fatalf("tras caducar el límite = %v", err)
	}
}

func TestAprobarLimiteYNormalizacion(t *testing.T) {
	svc, db, _ := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	pairing, err := svc.CreatePairing(ctx, "Móvil", ClientAndroid, "198.51.100.40")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	wrong := differentPairCode(pairing.Code)
	for attempt := 0; attempt < pairApproveLimit; attempt++ {
		if _, err := svc.ApprovePairing(ctx, userID, wrong, false, "198.51.100.40"); !errors.Is(err, ErrPairingNotFound) {
			t.Fatalf("intento %d = %v", attempt+1, err)
		}
	}
	if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, false, "198.51.100.40"); !errors.Is(err, ErrPairingRateLimited) {
		t.Fatalf("tras los fallos = %v", err)
	}
	if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, false, "198.51.100.41"); err != nil {
		t.Fatalf("otra IP debe poder aprobar: %v", err)
	}

	normalized := strings.ToLower(strings.ReplaceAll(pairing.Code, "-", ""))
	spaced := " " + normalized[:4] + " \t- " + normalized[4:] + " "
	if _, err := svc.ApprovePairing(ctx, userID, spaced, false, "198.51.100.41"); err != nil {
		t.Fatalf("código normalizado: %v", err)
	}
	invalid := "I" + normalized[1:]
	if _, err := svc.ApprovePairing(ctx, userID, invalid, false, "198.51.100.41"); !errors.Is(err, ErrPairingCodeInvalid) {
		t.Fatalf("carácter fuera del alfabeto = %v", err)
	}
}

func TestUsuarioDesactivadoOBorradoNoRecibeToken(t *testing.T) {
	t.Run("desactivado", func(t *testing.T) {
		svc, db, _ := newTestService(t)
		userID := createUser(t, db, "ana", "Clave-segura-1", "user")
		ctx := context.Background()
		pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroidTV, "198.51.100.50")
		if err != nil {
			t.Fatalf("CreatePairing: %v", err)
		}
		if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, true, "198.51.100.50"); err != nil {
			t.Fatalf("aprobar: %v", err)
		}
		if _, err := db.Exec(`UPDATE users SET disabled = 1 WHERE id = ?`, userID); err != nil {
			t.Fatalf("desactivar: %v", err)
		}
		if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.50"); !errors.Is(err, ErrPairingUserInactive) {
			t.Fatalf("token de usuario desactivado = %v", err)
		}
		var sessions int
		if err := db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id = ?`, userID).Scan(&sessions); err != nil {
			t.Fatalf("contar sesiones: %v", err)
		}
		if sessions != 0 {
			t.Fatalf("se creó una sesión para un usuario desactivado")
		}
	})

	t.Run("borrado", func(t *testing.T) {
		svc, db, _ := newTestService(t)
		userID := createUser(t, db, "ana", "Clave-segura-1", "user")
		ctx := context.Background()
		pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.51")
		if err != nil {
			t.Fatalf("CreatePairing: %v", err)
		}
		if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, true, "198.51.100.51"); err != nil {
			t.Fatalf("aprobar: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
			t.Fatalf("borrar usuario: %v", err)
		}
		var approvedUser sql.NullInt64
		if err := db.QueryRow(
			`SELECT approved_user_id FROM device_pairings WHERE pair_id = ?`, pairing.PairID,
		).Scan(&approvedUser); err != nil {
			t.Fatalf("leer vinculación huérfana: %v", err)
		}
		if approvedUser.Valid {
			t.Fatal("la vinculación debe quedar sin usuario aprobador")
		}
		if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.51"); !errors.Is(err, ErrPairingUserInactive) {
			t.Fatalf("token de usuario borrado = %v", err)
		}
	})
}

func TestSondeoDemasiadoRapido(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.60")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.60"); !errors.Is(err, ErrPairingPending) {
		t.Fatalf("primer sondeo = %v", err)
	}
	clock.Advance(time.Second)
	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.60"); !errors.Is(err, ErrPairingTooFast) {
		t.Fatalf("sondeo rápido = %v", err)
	}
	clock.Advance(pairingPollMin - time.Second)
	if _, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.60"); !errors.Is(err, ErrPairingPending) {
		t.Fatalf("sondeo en el intervalo = %v", err)
	}
}

func TestLimitesDeCreacionYGlobal(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()

	for attempt := 0; attempt < pairCreateLimit; attempt++ {
		if _, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.70"); err != nil {
			t.Fatalf("creación %d: %v", attempt+1, err)
		}
	}
	if _, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.70"); !errors.Is(err, ErrPairingRateLimited) {
		t.Fatalf("límite por IP = %v", err)
	}

	globalSvc, _, _ := newTestService(t)
	for attempt := 0; attempt < pairMaxPending; attempt++ {
		ip := fmt.Sprintf("192.0.2.%d", attempt+1)
		if _, err := globalSvc.CreatePairing(ctx, "TV", ClientAndroid, ip); err != nil {
			t.Fatalf("creación global %d: %v", attempt+1, err)
		}
	}
	if _, err := globalSvc.CreatePairing(ctx, "TV", ClientAndroid, "192.0.2.250"); !errors.Is(err, ErrPairingLimit) {
		t.Fatalf("límite global = %v", err)
	}
}

func TestCleanupPairings(t *testing.T) {
	svc, db, clock := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.80"); err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	clock.Advance(pairingTTL + time.Second)
	if err := svc.CleanupPairings(ctx); err != nil {
		t.Fatalf("CleanupPairings: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM device_pairings`).Scan(&count); err != nil {
		t.Fatalf("contar vinculaciones: %v", err)
	}
	if count != 0 {
		t.Fatalf("quedan %d vinculaciones caducadas", count)
	}
}

func TestCanjeConcurrenteSoloEntregaUnToken(t *testing.T) {
	svc, db, _ := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()
	pairing, err := svc.CreatePairing(ctx, "TV", ClientAndroid, "198.51.100.90")
	if err != nil {
		t.Fatalf("CreatePairing: %v", err)
	}
	if _, err := svc.ApprovePairing(ctx, userID, pairing.Code, true, "198.51.100.90"); err != nil {
		t.Fatalf("aprobar: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.RedeemPairing(ctx, pairing.PairID, pairing.Secret, "198.51.100.90")
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes, expired := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPairingExpired):
			expired++
		default:
			t.Fatalf("canje concurrente: %v", err)
		}
	}
	if successes != 1 || expired != 1 {
		t.Fatalf("canjes: %d correctos, %d caducados", successes, expired)
	}
	var sessions int
	if err := db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id = ? AND client = ?`,
		userID, ClientAndroid).Scan(&sessions); err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if sessions != 1 {
		t.Fatalf("sesiones creadas = %d", sessions)
	}
}

func TestVinculacionHTTP(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	loginStatus, token := httpLogin(t, server.URL, "ana", "Clave-segura-1")
	if loginStatus != http.StatusOK {
		t.Fatalf("login: %d", loginStatus)
	}

	status, _, body := pairingHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/auth/pair", "",
		map[string]string{"device_name": "TV", "client": "web"})
	if status != http.StatusBadRequest {
		t.Fatalf("cliente web = %d %s", status, body)
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/auth/pair", "",
		map[string]string{"device_name": "TV", "client": "androidtv"})
	if status != http.StatusCreated {
		t.Fatalf("crear vinculación = %d %s", status, body)
	}
	var created pairCreateResponse
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("respuesta de vinculación: %v", err)
	}
	if created.PairID == "" || created.Secret == "" || created.Code == "" ||
		created.VerificationPath != "/vincular" || created.ExpiresIn != 600 || created.Interval != 5 {
		t.Fatalf("respuesta de vinculación inesperada: %+v", created)
	}

	status, headers, body := pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/token", "", map[string]string{
			"pair_id": created.PairID, "secret": created.Secret,
		})
	if status != http.StatusPreconditionRequired {
		t.Fatalf("token pendiente = %d %s", status, body)
	}
	if headers.Get("Retry-After") != "5" {
		t.Fatalf("Retry-After = %q", headers.Get("Retry-After"))
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/approve", "",
		map[string]any{"code": created.Code, "confirm": false})
	if status != http.StatusUnauthorized {
		t.Fatalf("aprobar sin sesión = %d %s", status, body)
	}

	normalized := strings.ToLower(created.Code)
	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/approve", token,
		map[string]any{"code": normalized, "confirm": false})
	if status != http.StatusOK {
		t.Fatalf("vista previa = %d %s", status, body)
	}
	var preview pairApprovePreviewResponse
	if err := json.Unmarshal(body, &preview); err != nil {
		t.Fatalf("vista previa: %v", err)
	}
	if preview.DeviceName != "TV" || preview.Client != ClientAndroidTV || preview.ExpiresAt.IsZero() {
		t.Fatalf("vista previa inesperada: %+v", preview)
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/approve", token,
		map[string]any{"code": created.Code, "confirm": true})
	if status != http.StatusOK {
		t.Fatalf("aprobar = %d %s", status, body)
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/token", "", map[string]string{
			"pair_id": created.PairID, "secret": created.Secret,
		})
	if status != http.StatusOK {
		t.Fatalf("canjear = %d %s", status, body)
	}
	var redeemed pairTokenResponse
	if err := json.Unmarshal(body, &redeemed); err != nil {
		t.Fatalf("respuesta de token: %v", err)
	}
	if redeemed.Token == "" || redeemed.SessionID == 0 ||
		redeemed.User.Username != "ana" || redeemed.User.Role != "user" {
		t.Fatalf("token de vinculación inesperado: %+v", redeemed)
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/token", "", map[string]string{
			"pair_id": created.PairID, "secret": created.Secret,
		})
	if status != http.StatusGone {
		t.Fatalf("segundo canje = %d %s", status, body)
	}

	status, _, body = pairingHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/auth/pair/approve", token,
		map[string]any{"code": created.Code, "confirm": false})
	if status != http.StatusNotFound || !strings.Contains(string(body), "código no válido o caducado") {
		t.Fatalf("aprobar usado = %d %s", status, body)
	}
}

func pairingHTTPRequest(
	t *testing.T,
	serverURL, method, path, token string,
	body any,
) (int, http.Header, []byte) {
	t.Helper()
	return pairingHTTPDo(t, serverURL, method, path, token, body)
}

func pairingHTTPDo(
	t *testing.T,
	serverURL, method, path, token string,
	body any,
) (int, http.Header, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, serverURL+path, reader)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, resp.Header, data
}
