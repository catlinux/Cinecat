package library

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

type castCreditDTO struct {
	Name      string `json:"name"`
	Character string `json:"character"`
	PhotoURL  string `json:"photo_url"`
}

type crewCreditDTO struct {
	Name     string `json:"name"`
	Job      string `json:"job"`
	PhotoURL string `json:"photo_url"`
}

type creditsDTO struct {
	Cast []castCreditDTO `json:"cast"`
	Crew []crewCreditDTO `json:"crew"`
}

type storedCastCredit struct {
	Name      string `json:"name"`
	Character string `json:"character"`
	Photo     string `json:"photo"`
}

type storedCrewCredit struct {
	Name  string `json:"name"`
	Job   string `json:"job"`
	Photo string `json:"photo"`
}

type storedCredits struct {
	Cast []storedCastCredit `json:"cast"`
	Crew []storedCrewCredit `json:"crew"`
}

func emptyCreditsDTO() creditsDTO {
	return creditsDTO{Cast: make([]castCreditDTO, 0), Crew: make([]crewCreditDTO, 0)}
}

func creditsJSON(credits metadata.Credits) string {
	stored := storedCredits{
		Cast: make([]storedCastCredit, 0, len(credits.Cast)),
		Crew: make([]storedCrewCredit, 0, len(credits.Crew)),
	}
	for _, credit := range credits.Cast {
		if strings.TrimSpace(credit.Name) == "" {
			continue
		}
		stored.Cast = append(stored.Cast, storedCastCredit{
			Name: credit.Name, Character: credit.Character, Photo: credit.PhotoURL,
		})
	}
	for _, credit := range credits.Crew {
		if strings.TrimSpace(credit.Name) == "" {
			continue
		}
		stored.Crew = append(stored.Crew, storedCrewCredit{
			Name: credit.Name, Job: credit.Job, Photo: credit.PhotoURL,
		})
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return `{"cast":[],"crew":[]}`
	}
	return string(data)
}

func decodeCredits(raw sql.NullString) creditsDTO {
	result := emptyCreditsDTO()
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return result
	}
	var stored storedCredits
	if err := json.Unmarshal([]byte(raw.String), &stored); err != nil {
		return result
	}
	for _, credit := range stored.Cast {
		if strings.TrimSpace(credit.Name) == "" {
			continue
		}
		result.Cast = append(result.Cast, castCreditDTO{
			Name: credit.Name, Character: credit.Character, PhotoURL: credit.Photo,
		})
	}
	for _, credit := range stored.Crew {
		if strings.TrimSpace(credit.Name) == "" {
			continue
		}
		result.Crew = append(result.Crew, crewCreditDTO{
			Name: credit.Name, Job: credit.Job, PhotoURL: credit.Photo,
		})
	}
	return result
}

// similarCastNames devuelve los diez primeros nombres del reparto guardado.
func similarCastNames(raw sql.NullString, fallback sql.NullString) []string {
	if raw.Valid && strings.TrimSpace(raw.String) != "" {
		var stored storedCredits
		if err := json.Unmarshal([]byte(raw.String), &stored); err == nil {
			names := make([]string, 0, 10)
			for _, credit := range stored.Cast {
				if len(names) == 10 {
					break
				}
				if strings.TrimSpace(credit.Name) != "" {
					names = append(names, credit.Name)
				}
			}
			if len(names) > 0 {
				return names
			}
		}
	}
	names := decodeStringArray(fallback)
	if len(names) > 10 {
		names = names[:10]
	}
	return names
}
