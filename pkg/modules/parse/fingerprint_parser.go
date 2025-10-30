// pkg/modules/parse/fingerprint_parser.go
package parse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/fingerprint"
	"github.com/pentora-ai/pentora/pkg/modules/scan"
)

const (
	fingerprintParserModuleID          = "fingerprint-parser-instance"
	fingerprintParserModuleName        = "fingerprint-parser"
	fingerprintParserModuleDescription = "Matches service banners with fingerprint catalog entries."
	fingerprintParserModuleVersion     = "0.1.0"
	fingerprintParserModuleAuthor      = "Pentora Team"
)

// FingerprintParsedInfo represents structured fingerprint output.
type FingerprintParsedInfo struct {
	Target      string  `json:"target"`
	Port        int     `json:"port"`
	Protocol    string  `json:"protocol,omitempty"`
	Product     string  `json:"product,omitempty"`
	Vendor      string  `json:"vendor,omitempty"`
	Version     string  `json:"version,omitempty"`
	CPE         string  `json:"cpe,omitempty"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description,omitempty"`
	SourceProbe string  `json:"source_probe,omitempty"`
}

// FingerprintParserModule consumes banner results and produces fingerprint matches.
type FingerprintParserModule struct {
	meta engine.ModuleMetadata
}

func newFingerprintParserModule() *FingerprintParserModule {
	return &FingerprintParserModule{
		meta: engine.ModuleMetadata{
			ID:          fingerprintParserModuleID,
			Name:        fingerprintParserModuleName,
			Description: fingerprintParserModuleDescription,
			Version:     fingerprintParserModuleVersion,
			Type:        engine.ParseModuleType,
			Author:      fingerprintParserModuleAuthor,
			Tags:        []string{"parser", "fingerprint"},
			Consumes: []engine.DataContractEntry{
				{
					Key:          "service.banner.tcp",
					DataTypeName: "scan.BannerGrabResult",
					Cardinality:  engine.CardinalityList,
					IsOptional:   true,
					Description:  "List of raw TCP banners captured from the service-banner module.",
				},
			},
			Produces: []engine.DataContractEntry{
				{
					Key:          "service.fingerprint.details",
					DataTypeName: "parse.FingerprintParsedInfo",
					Cardinality:  engine.CardinalityList,
					Description:  "Fingerprint matches derived from service banners.",
				},
			},
		},
	}
}

// Metadata returns the metadata information for the FingerprintParserModule.
// It implements the engine.Module interface.
func (m *FingerprintParserModule) Metadata() engine.ModuleMetadata { return m.meta }

func (m *FingerprintParserModule) Init(instanceID string, _ map[string]interface{}) error {
	m.meta.ID = instanceID
	initLogger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()
	initLogger.Debug().Msg("Fingerprint parser initialized")
	return nil
}

func (m *FingerprintParserModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()
	raw, ok := inputs["service.banner.tcp"]
	if !ok {
		return nil
	}

	bannerList, listOk := raw.([]interface{})
	if !listOk {
		logger.Warn().Type("input_type", raw).Msg("'service.banner.tcp' not a list, skipping fingerprint parser")
		return nil
	}

	resolver := fingerprint.GetFingerprintResolver()
	matches := 0

	for _, item := range bannerList {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		banner, castOk := item.(scan.BannerGrabResult)
		if !castOk {
			continue
		}

		seenResponses := make(map[string]struct{})
		seenMatches := make(map[string]struct{})

		for _, candidate := range gatherBannerCandidates(banner) {
			response := strings.TrimSpace(candidate.Response)
			if response == "" {
				continue
			}
			if _, exists := seenResponses[response]; exists {
				continue
			}
			seenResponses[response] = struct{}{}

			protocolHint := strings.ToLower(candidate.Protocol)
			if protocolHint == "" {
				protocolHint = strings.ToLower(banner.Protocol)
			}
			if protocolHint == "" {
				protocolHint = fingerprintProtocolHint(banner.Port, response)
			}

			result, err := resolver.Resolve(ctx, fingerprint.Input{
				Protocol:    protocolHint,
				Banner:      response,
				Port:        banner.Port,
				ServiceHint: "",
			})
			if err != nil || result.Product == "" {
				continue
			}

			matchKey := fmt.Sprintf("%s|%s|%s", result.Product, result.Version, protocolHint)
			if _, exists := seenMatches[matchKey]; exists {
				continue
			}
			seenMatches[matchKey] = struct{}{}

			parsed := FingerprintParsedInfo{
				Target:      banner.IP,
				Port:        banner.Port,
				Protocol:    protocolHint,
				Product:     result.Product,
				Vendor:      result.Vendor,
				Version:     result.Version,
				CPE:         result.CPE,
				Confidence:  result.Confidence,
				Description: result.Description,
				SourceProbe: candidate.ProbeID,
			}

			outputChan <- engine.ModuleOutput{
				FromModuleName: m.meta.ID,
				DataKey:        m.meta.Produces[0].Key,
				Data:           parsed,
				Timestamp:      time.Now(),
				Target:         banner.IP,
			}
			matches++
		}
	}

	logger.Info().Int("matches", matches).Msg("Fingerprint parsing completed")
	return nil
}

type bannerCandidate struct {
	Response string
	Protocol string
	ProbeID  string
}

func gatherBannerCandidates(banner scan.BannerGrabResult) []bannerCandidate {
	candidates := make([]bannerCandidate, 0, len(banner.Evidence)+1)

	if trimmed := strings.TrimSpace(banner.Banner); trimmed != "" {
		candidates = append(candidates, bannerCandidate{
			Response: trimmed,
			Protocol: banner.Protocol,
			ProbeID:  "tcp-passive",
		})
	}

	for _, obs := range banner.Evidence {
		resp := strings.TrimSpace(obs.Response)
		if resp == "" {
			continue
		}
		protocol := obs.Protocol
		if protocol == "" {
			protocol = banner.Protocol
		}
		candidates = append(candidates, bannerCandidate{
			Response: resp,
			Protocol: protocol,
			ProbeID:  obs.ProbeID,
		})
	}

	return candidates
}

func fingerprintProtocolHint(_ int, banner string) string {
	banner = strings.ToLower(banner)
	switch {
	case strings.HasPrefix(banner, "ssh-"):
		return "ssh"
	case strings.Contains(banner, "http/") || strings.Contains(banner, "server:"):
		return "http"
	case strings.Contains(banner, "smtp"):
		return "smtp"
	case strings.Contains(banner, "ftp"):
		return "ftp"
	case strings.Contains(banner, "mysql"):
		return "mysql"
	}
	return ""
}

func fingerprintParserModuleFactory() engine.Module {
	return newFingerprintParserModule()
}

func init() {
	engine.RegisterModuleFactory(fingerprintParserModuleName, fingerprintParserModuleFactory)
}
