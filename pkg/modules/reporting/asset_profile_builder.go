// pkg/modules/reporting/asset_profile_builder.go
package reporting // veya uygun bir paket adı

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/discovery" // For ICMPPingDiscoveryResult, TCPPortDiscoveryResult
	"github.com/pentora-ai/pentora/pkg/modules/parse"     // For HTTPParsedInfo, SSHParsedInfo
	"github.com/pentora-ai/pentora/pkg/modules/scan"      // For BannerScanResult
	"github.com/pentora-ai/pentora/pkg/utils"
	"github.com/rs/zerolog/log"
)

const (
	assetProfileBuilderModuleTypeName = "asset-profile-builder"
)

// AssetProfileBuilderConfig (şu an için boş, ileride eklenebilir)
type AssetProfileBuilderConfig struct{}

// AssetProfileBuilderModule implements the engine.Module interface.
type AssetProfileBuilderModule struct {
	meta   engine.ModuleMetadata
	config AssetProfileBuilderConfig
}

func newAssetProfileBuilderModule() *AssetProfileBuilderModule {
	return &AssetProfileBuilderModule{
		meta: engine.ModuleMetadata{
			Name:        assetProfileBuilderModuleTypeName,
			Version:     "0.1.0",
			Description: "Aggregates all scan data into comprehensive asset profiles.",
			Type:        engine.ReportingModuleType, // veya OrchestrationModuleType
			Author:      "Pentora Team",
			Tags:        []string{"reporting", "aggregation", "asset-profile"},
			Consumes: []engine.DataContractEntry{ // Bu modül birçok şeyi tüketir
				// Planner bu anahtarları DataContext'ten alıp bu modülün input'una verir.
				// Veya bu modül doğrudan DataContext'in tamamını alıp kendi içinde filtreleyebilir.
				// Şimdilik spesifik anahtarlar varsayalım:
				{Key: "config.targets", DataTypeName: "[]string", Cardinality: engine.CardinalitySingle, IsOptional: true},
				{Key: "discovery.live_hosts", DataTypeName: "discovery.ICMPPingDiscoveryResult", Cardinality: engine.CardinalityList, IsOptional: true},    // DataContext'te []interface{}{ICMPPingDiscoveryResult}
				{Key: "discovery.open_tcp_ports", DataTypeName: "discovery.TCPPortDiscoveryResult", Cardinality: engine.CardinalityList, IsOptional: true}, // []interface{}{TCPPortDiscoveryResult1, TCPResult2}
				{Key: "service.banner.tcp", DataTypeName: "scan.BannerScanResult", Cardinality: engine.CardinalityList, IsOptional: true},                  // []interface{}{BannerResult1, BannerResult2}
				{Key: "service.http.details", DataTypeName: "parse.HTTPParsedInfo", Cardinality: engine.CardinalityList, IsOptional: true},                 // []interface{}{HTTPParsedInfo1, ...}
				{Key: "service.ssh.details", DataTypeName: "parse.SSHParsedInfo", Cardinality: engine.CardinalityList, IsOptional: true},                   // []interface{}{SSHParsedInfo1, ...}
				// Zafiyetler için genel bir pattern veya planner'ın dinamik olarak eklemesi gerekebilir:
				// Örnek: {Key: "vulnerability.*", DataTypeName: "types.VulnerabilityFinding", Cardinality: engine.CardinalityList, IsOptional: true},
			},
			Produces: []engine.DataContractEntry{
				{Key: "asset.profiles", DataTypeName: "[]types.AssetProfile", Cardinality: engine.CardinalitySingle}, // Tek bir liste üretir
			},
			ConfigSchema: map[string]engine.ParameterDefinition{},
		},
		config: AssetProfileBuilderConfig{},
	}
}

func (m *AssetProfileBuilderModule) Metadata() engine.ModuleMetadata { return m.meta }

func (m *AssetProfileBuilderModule) Init(instanceID string, configMap map[string]interface{}) error {
	m.meta.ID = instanceID
	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()
	logger.Debug().Msg("Initializing AssetProfileBuilderModule")
	// No specific config to parse for now
	return nil
}

func (m *AssetProfileBuilderModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()
	logger.Info().Msg("Starting asset profile aggregation")
	logger.Debug().Interface("received_inputs_for_aggregation", inputs).Msg("Full inputs")

	// Helper to safely get and cast data from inputs
	// Tüketilen her anahtarın []interface{} listesi olarak geldiğini varsayıyoruz (modül çıktıları için)
	// veya doğrudan tip (initialInputs için). DataContext ve Orchestrator'daki Get/Set mantığına bağlı.
	// Bir önceki konuşmamızdaki DataContext.SetInitial ve AddModuleOutput ayrımına göre:

	var initialTargets []string
	if rawInitialTargets, ok := inputs["config.targets"]; ok {
		if casted, castOk := rawInitialTargets.([]string); castOk { // SetInitial doğrudan saklar
			initialTargets = casted
		} else if rawInitialTargets != nil {
			logger.Warn().Type("type", rawInitialTargets).Msg("config.targets input has unexpected type")
		}
	}

	liveHostResults := []discovery.ICMPPingDiscoveryResult{}
	if rawLiveHosts, ok := inputs["discovery.live_hosts"]; ok {
		if list, listOk := rawLiveHosts.([]interface{}); listOk {
			for _, item := range list {
				if casted, castOk := item.(discovery.ICMPPingDiscoveryResult); castOk {
					liveHostResults = append(liveHostResults, casted)
				} // else log cast error
			}
		} // else log not a list error
	}

	openTCPPortResults := []discovery.TCPPortDiscoveryResult{}
	if rawOpenTCPPorts, ok := inputs["discovery.open_tcp_ports"]; ok {
		if list, listOk := rawOpenTCPPorts.([]interface{}); listOk {
			for _, item := range list {
				if casted, castOk := item.(discovery.TCPPortDiscoveryResult); castOk {
					openTCPPortResults = append(openTCPPortResults, casted)
				}
			}
		}
	}

	bannerResults := []scan.BannerGrabResult{}
	if rawBanners, ok := inputs["service.banner.tcp"]; ok { // veya service.banner.raw
		if list, listOk := rawBanners.([]interface{}); listOk {
			for _, item := range list {
				if casted, castOk := item.(scan.BannerGrabResult); castOk {
					bannerResults = append(bannerResults, casted)
				}
			}
		}
	}

	httpDetailsResults := []parse.HTTPParsedInfo{}
	if rawHTTP, ok := inputs["service.http.details"]; ok {
		if list, listOk := rawHTTP.([]interface{}); listOk {
			for _, item := range list {
				if casted, castOk := item.(parse.HTTPParsedInfo); castOk {
					httpDetailsResults = append(httpDetailsResults, casted)
				}
			}
		}
	}

	sshDetailsResults := []parse.SSHParsedInfo{}
	if rawSSH, ok := inputs["service.ssh.details"]; ok {
		if list, listOk := rawSSH.([]interface{}); listOk {
			for _, item := range list {
				if casted, castOk := item.(parse.SSHParsedInfo); castOk {
					sshDetailsResults = append(sshDetailsResults, casted)
				}
			}
		}
	}

	// TODO: Zafiyetleri de benzer şekilde topla.
	// Zafiyet modüllerinin çıktılarının types.VulnerabilityFinding veya benzeri bir struct olması beklenir.
	// Ve DataContext'te "instance_id.vulnerability.<type>.<vuln_id>" gibi anahtarlarla saklanabilirler.
	// Bu modül, tüm bu anahtarları tarayarak veya belirli bir pattern'e uyanları alarak zafiyetleri toplar.
	allVulnerabilities := make(map[string][]engine.VulnerabilityFinding) // Key: targetIP:port

	// inputs map'i üzerinde dönerek vulnerability.* anahtarlarını bul (veya DataContext'i al)
	for key, data := range inputs { // Bu basit bir yaklaşım, DataContext'in tamamına erişim daha iyi olabilir.
		if strings.Contains(key, "vulnerability.") { // instanceID.vulnerability.ssh.default_creds gibi
			if vulnList, listOk := data.([]interface{}); listOk {
				for _, item := range vulnList {
					if vuln, castOk := item.(engine.VulnerabilityFinding); castOk { // Varsayım: Zafiyet modülleri bu tipi üretir
						targetPortKey := fmt.Sprintf("%s:%d", vuln.Target /*modülün bu alanı doldurması lazım*/, vuln.Port /*modülün bu alanı doldurması lazım*/)
						allVulnerabilities[targetPortKey] = append(allVulnerabilities[targetPortKey], vuln)
					}
				}
			}
		}
	}

	// Ana veri işleme ve birleştirme mantığı
	finalAssetProfiles := []engine.AssetProfile{}
	processedTargets := make(map[string]*engine.AssetProfile) // IP adresine göre AssetProfile tutar

	// 1. Canlı hostlardan AssetProfile'ları başlat
	for _, icmpResult := range liveHostResults {
		for _, liveIP := range icmpResult.LiveHosts {
			if _, exists := processedTargets[liveIP]; !exists {
				now := time.Now()
				profile := &engine.AssetProfile{
					Target:              liveIP, // Başlangıçta IP'yi target olarak al, sonra hostname eklenebilir
					ResolvedIPs:         map[string]time.Time{liveIP: now},
					IsAlive:             true,
					ScanStartTime:       now, // Bu, bu modülün başlangıç zamanı, daha iyisi DAG başlangıcı
					LastObservationTime: now,
					OpenPorts:           make(map[string][]engine.PortProfile),
				}
				processedTargets[liveIP] = profile
				finalAssetProfiles = append(finalAssetProfiles, *profile) // Slice'a eklerken değerini kopyala
			} else {
				processedTargets[liveIP].IsAlive = true
				processedTargets[liveIP].LastObservationTime = time.Now()
			}
		}
	}

	// Eğer canlı host bilgisi yoksa, initialTargets'ı kullan (ping kapalıysa veya yanıt yoksa)
	if len(liveHostResults) == 0 {
		expandedInitialTargets := utils.ParseAndExpandTargets(initialTargets) // utils'dan
		for _, target := range expandedInitialTargets {
			if _, exists := processedTargets[target]; !exists {
				now := time.Now()
				profile := &engine.AssetProfile{
					Target:              target,
					ResolvedIPs:         map[string]time.Time{target: now},
					IsAlive:             false, // Ping ile doğrulanmadı
					ScanStartTime:       now,
					LastObservationTime: now,
					OpenPorts:           make(map[string][]engine.PortProfile),
				}
				processedTargets[target] = profile
				finalAssetProfiles = append(finalAssetProfiles, *profile)
			}
		}
	}

	// 2. Her bir AssetProfile'ı güncelle (referans üzerinden)
	for i := range finalAssetProfiles {
		asset := &finalAssetProfiles[i] // Referans alarak güncelleme yapabilmek için
		targetIP := asset.Target        // Veya ResolvedIPs'ten biri (şimdilik Target'ı IP kabul edelim)

		assetOpenPorts := []engine.PortProfile{}

		// Açık TCP Portlarını işle
		for _, tcpResult := range openTCPPortResults {
			if tcpResult.Target == targetIP {
				for _, portNum := range tcpResult.OpenPorts {
					portProfile := engine.PortProfile{
						PortNumber: portNum,
						Protocol:   "tcp",
						Status:     "open",
						Service:    engine.ServiceDetails{},
					}

					// Bu porta ait banner'ı bul
					for _, banner := range bannerResults {
						if banner.IP == targetIP && banner.Port == portNum {
							portProfile.Service.RawBanner = banner.Banner
							portProfile.Service.IsTLS = banner.IsTLS
							break
						}
					}

					// Bu porta ait parse edilmiş HTTP detaylarını bul
					for _, httpDetail := range httpDetailsResults {
						if httpDetail.Target == targetIP && httpDetail.Port == portNum {
							portProfile.Service.Name = "http" // Veya httpDetail.ServerProduct
							if httpDetail.ServerProduct != "" {
								portProfile.Service.Product = httpDetail.ServerProduct
							} else {
								portProfile.Service.Product = "HTTP" // Genel
							}
							portProfile.Service.Version = httpDetail.ServerVersion
							if portProfile.Service.ParsedAttributes == nil {
								portProfile.Service.ParsedAttributes = make(map[string]interface{})
							}
							portProfile.Service.ParsedAttributes["http_status_code"] = httpDetail.StatusCode
							portProfile.Service.ParsedAttributes["http_version"] = httpDetail.HTTPVersion
							portProfile.Service.ParsedAttributes["html_title"] = httpDetail.HTMLTitle
							portProfile.Service.ParsedAttributes["content_type"] = httpDetail.ContentType
							portProfile.Service.ParsedAttributes["headers"] = httpDetail.Headers
							portProfile.Service.Scheme = httpDetail.Scheme
							break
						}
					}
					// Bu porta ait parse edilmiş SSH detaylarını bul
					for _, sshDetail := range sshDetailsResults {
						if sshDetail.Target == targetIP && sshDetail.Port == portNum {
							portProfile.Service.Name = sshDetail.ProtocolName
							portProfile.Service.Product = sshDetail.Software
							portProfile.Service.Version = sshDetail.SoftwareVersion
							if portProfile.Service.ParsedAttributes == nil {
								portProfile.Service.ParsedAttributes = make(map[string]interface{})
							}
							portProfile.Service.ParsedAttributes["ssh_protocol_version"] = sshDetail.SSHVersion
							portProfile.Service.ParsedAttributes["ssh_full_version_info"] = sshDetail.VersionInfo
							break
						}
					}

					// Bu porta ait zafiyetleri bul
					targetPortKey := fmt.Sprintf("%s:%d", targetIP, portNum)
					if vulns, found := allVulnerabilities[targetPortKey]; found {
						portProfile.Vulnerabilities = vulns
						asset.TotalVulnerabilities += len(vulns)
					}

					assetOpenPorts = append(assetOpenPorts, portProfile)
				}
			}
		}
		asset.OpenPorts[targetIP] = assetOpenPorts // Haritaya ekle
		asset.LastObservationTime = time.Now()
	}

	// asset.profiles'ı ModuleOutput olarak gönder
	outputChan <- engine.ModuleOutput{
		FromModuleName: m.meta.ID,
		DataKey:        m.meta.Produces[0].Key, // "asset.profiles"
		Data:           finalAssetProfiles,     // Bu []types.AssetProfile tipinde olmalı
		Timestamp:      time.Now(),
	}

	logger.Info().Int("profile_count", len(finalAssetProfiles)).Msg("Asset profile aggregation completed")
	return nil
}

func AssetProfileBuilderModuleFactory() engine.Module {
	return newAssetProfileBuilderModule()
}

func init() {
	engine.RegisterModuleFactory(assetProfileBuilderModuleTypeName, AssetProfileBuilderModuleFactory)
}
