package auth

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type pairCreateRequest struct {
	DeviceName string `json:"device_name"`
	Client     string `json:"client"`
}

type pairCreateResponse struct {
	PairID           string `json:"pair_id"`
	Secret           string `json:"secret"`
	Code             string `json:"code"`
	VerificationPath string `json:"verification_path"`
	ExpiresIn        int    `json:"expires_in"`
	Interval         int    `json:"interval"`
}

type pairApproveRequest struct {
	Code    string `json:"code"`
	Confirm bool   `json:"confirm"`
}

type pairApprovePreviewResponse struct {
	DeviceName string    `json:"device_name"`
	Client     string    `json:"client"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type pairApproveResponse struct {
	DeviceName string `json:"device_name"`
	Client     string `json:"client"`
}

type pairTokenRequest struct {
	PairID string `json:"pair_id"`
	Secret string `json:"secret"`
}

type pairUserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type pairTokenResponse struct {
	Token     string           `json:"token"`
	ExpiresAt time.Time        `json:"expires_at"`
	SessionID int64            `json:"session_id"`
	User      pairUserResponse `json:"user"`
}

// RegisterPairingRoutes monta la vinculación por código de dispositivos
// nativos.
func (s *Service) RegisterPairingRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.HandleFunc("POST /api/v1/auth/pair", s.handlePairCreate)
	mux.HandleFunc("POST /api/v1/auth/pair/approve", s.handlePairApprove)
	mux.HandleFunc("POST /api/v1/auth/pair/token", s.handlePairToken)
}

func (s *Service) handlePairCreate(w http.ResponseWriter, r *http.Request) {
	var req pairCreateRequest
	if !decodePairRequest(w, r, &req) {
		return
	}
	pairing, err := s.CreatePairing(r.Context(), req.DeviceName, req.Client,
		ClientIP(r, s.trusted))
	switch {
	case errors.Is(err, ErrInvalidClient):
		writeError(w, http.StatusBadRequest, "cliente no válido")
		return
	case errors.Is(err, ErrPairingRateLimited):
		writeError(w, http.StatusTooManyRequests, "demasiadas vinculaciones")
		return
	case errors.Is(err, ErrPairingLimit):
		writeError(w, http.StatusServiceUnavailable, "demasiadas vinculaciones pendientes")
		return
	case err != nil:
		log.Printf("auth: crear vinculación: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusCreated, pairCreateResponse{
		PairID:           pairing.PairID,
		Secret:           pairing.Secret,
		Code:             pairing.Code,
		VerificationPath: pairingPath,
		ExpiresIn:        int(pairingTTL / time.Second),
		Interval:         int(pairingPollEvery / time.Second),
	})
}

func (s *Service) handlePairApprove(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSessionRequest(w, r)
	if !ok {
		return
	}
	var req pairApproveRequest
	if !decodePairRequest(w, r, &req) {
		return
	}
	preview, err := s.ApprovePairing(r.Context(), user.ID, req.Code, req.Confirm,
		ClientIP(r, s.trusted))
	switch {
	case errors.Is(err, ErrPairingCodeInvalid):
		writeError(w, http.StatusBadRequest, "código no válido")
		return
	case errors.Is(err, ErrPairingRateLimited):
		writeError(w, http.StatusTooManyRequests, "demasiados intentos")
		return
	case errors.Is(err, ErrPairingNotFound):
		writeError(w, http.StatusNotFound, "código no válido o caducado")
		return
	case err != nil:
		log.Printf("auth: aprobar vinculación: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}

	if !req.Confirm {
		writeJSON(w, http.StatusOK, pairApprovePreviewResponse{
			DeviceName: preview.DeviceName,
			Client:     preview.Client,
			ExpiresAt:  preview.ExpiresAt,
		})
		return
	}
	writeJSON(w, http.StatusOK, pairApproveResponse{
		DeviceName: preview.DeviceName,
		Client:     preview.Client,
	})
}

func (s *Service) handlePairToken(w http.ResponseWriter, r *http.Request) {
	var req pairTokenRequest
	if !decodePairRequest(w, r, &req) {
		return
	}
	result, err := s.RedeemPairing(r.Context(), req.PairID, req.Secret,
		ClientIP(r, s.trusted))
	switch {
	case errors.Is(err, ErrPairingTooFast):
		writeError(w, http.StatusTooManyRequests, "demasiado rápido")
		return
	case errors.Is(err, ErrPairingRateLimited):
		writeError(w, http.StatusTooManyRequests, "demasiados intentos")
		return
	case errors.Is(err, ErrPairingPending):
		w.Header().Set("Retry-After", strconv.Itoa(int(pairingPollEvery/time.Second)))
		writeError(w, http.StatusPreconditionRequired, "vinculación pendiente")
		return
	case errors.Is(err, ErrPairingNotFound):
		writeError(w, http.StatusNotFound, "vinculación no válida")
		return
	case errors.Is(err, ErrPairingExpired), errors.Is(err, ErrPairingUserInactive):
		writeError(w, http.StatusGone, "vinculación caducada")
		return
	case err != nil:
		log.Printf("auth: entregar vinculación: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, pairTokenResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		SessionID: result.SessionID,
		User: pairUserResponse{
			Username: result.Username,
			Role:     result.Role,
		},
	})
}

func decodePairRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := decodeJSON(w, r, dst); err != nil {
		if errors.Is(err, errBodyTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "el cuerpo de la petición supera el límite de 1 MiB")
			return false
		}
		writeError(w, http.StatusBadRequest, "petición no válida")
		return false
	}
	return true
}
