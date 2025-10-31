package fingerprint

import "testing"

func TestProbeCatalog_ProbesFor_Combinations(t *testing.T) {
    catalog := ProbeCatalog{Groups: []ProbeGroup{
        { // requires port 80, any protocol
            ID: "g1", PortHints: []int{80}, Probes: []ProbeSpec{{ID: "p1", Protocol: "tcp", Payload: "X"}},
        },
        { // requires http hint, no port hint
            ID: "g2", ProtocolHints: []string{"HTTP"}, Probes: []ProbeSpec{{ID: "p2", Protocol: "tcp", Payload: "Y"}},
        },
        { // both hints and includes/excludes
            ID: "g3", PortHints: []int{443}, ProtocolHints: []string{"https"}, Probes: []ProbeSpec{{ID: "p3", Protocol: "tcp", Payload: "Z", PortInclude: []int{443}, PortExclude: []int{8443}}},
        },
    }}

    // nil catalog safety
    var nilCat *ProbeCatalog
    if out := nilCat.ProbesFor(80, nil); out != nil { t.Fatalf("expected nil for nil catalog") }

    // port 80, no hints -> g1 matches (port hint present), g2 does not (needs http hint)
    out := catalog.ProbesFor(80, nil)
    if len(out) != 1 || out[0].ID != "p1" { t.Fatalf("unexpected for port80 no hints: %#v", out) }

    // port 25, http hint -> g2 matches via protocol hint even without port hints
    out = catalog.ProbesFor(25, []string{"http"})
    if len(out) != 1 || out[0].ID != "p2" { t.Fatalf("unexpected for port25 http hint: %#v", out) }

    // port 443 with https hint -> g3 matches; include/exclude allow 443
    out = catalog.ProbesFor(443, []string{"HTTPS"})
    if len(out) != 1 || out[0].ID != "p3" { t.Fatalf("unexpected for port443 https: %#v", out) }

    // port 8443 with https hint -> group matches but probe excludes this port
    out = catalog.ProbesFor(8443, []string{"https"})
    if len(out) != 0 { t.Fatalf("expected exclude to filter out, got %#v", out) }
}

func TestProbeCatalog_NormalizeHints_And_ContainsInt(t *testing.T) {
    // normalize lowercases and skips empties
    m := normalizeHints([]string{"HTTP", "", "Ssh"})
    if _, ok := m["http"]; !ok { t.Fatalf("expected http present") }
    if _, ok := m["ssh"]; !ok { t.Fatalf("expected ssh present") }
    if len(m) != 2 { t.Fatalf("unexpected len: %d", len(m)) }

    // containsInt true/false
    if !containsInt([]int{1,2,3}, 2) { t.Fatalf("containsInt should be true") }
    if containsInt([]int{1,2,3}, 4) { t.Fatalf("containsInt should be false") }
}

func TestProbeCatalog_Validate_Errors(t *testing.T) {
    var nilCat *ProbeCatalog
    if err := nilCat.Validate(); err == nil { t.Fatalf("expected error for nil catalog") }

    // missing group id
    c := ProbeCatalog{Groups: []ProbeGroup{{ID: "", Probes: []ProbeSpec{{ID: "p", Protocol: "x", Payload: "y"}}}}}
    if err := c.Validate(); err == nil { t.Fatalf("expected missing group id error") }

    // group with no probes
    c = ProbeCatalog{Groups: []ProbeGroup{{ID: "g", Probes: nil}}}
    if err := c.Validate(); err == nil { t.Fatalf("expected no probes error") }

    // probe missing id
    c = ProbeCatalog{Groups: []ProbeGroup{{ID: "g", Probes: []ProbeSpec{{ID: "", Protocol: "x", Payload: "y"}}}}}
    if err := c.Validate(); err == nil { t.Fatalf("expected probe missing id error") }

    // probe missing protocol
    c = ProbeCatalog{Groups: []ProbeGroup{{ID: "g", Probes: []ProbeSpec{{ID: "p", Protocol: "", Payload: "y"}}}}}
    if err := c.Validate(); err == nil { t.Fatalf("expected probe missing protocol error") }

    // probe missing payload
    c = ProbeCatalog{Groups: []ProbeGroup{{ID: "g", Probes: []ProbeSpec{{ID: "p", Protocol: "x", Payload: ""}}}}}
    if err := c.Validate(); err == nil { t.Fatalf("expected probe missing payload error") }
}

