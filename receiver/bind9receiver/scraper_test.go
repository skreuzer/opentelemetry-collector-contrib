package bind9receiver

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bind9receiver/internal/metadata"
)

func TestScraper(t *testing.T) {
	srv := newMockBIND9Server(t)
	defer srv.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = srv.URL
	require.NoError(t, xconfmap.Validate(cfg))

	scraper := newBind9Scraper(receivertest.NewNopSettings(metadata.Type), cfg)

	err := scraper.start(t.Context(), componenttest.NewNopHost())
	require.NoError(t, err)

	actualMetrics, err := scraper.scrape(t.Context())
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "scraper", "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreTimestamp(),
		pmetrictest.IgnoreMetricsOrder()))
}

func newMockBIND9Server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/json/v1/server":
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "boot-time":"2026-04-12T16:36:58.355Z",
  "config-time":"2026-04-12T16:36:58.385Z",
  "current-time":"2026-04-12T17:26:50.924Z",
  "version":"9.18.39",
  "opcodes":{
    "QUERY":100,
    "IQUERY":0,
    "STATUS":0,
    "RESERVED3":0,
    "NOTIFY":5,
    "UPDATE":0
  },
  "rcodes":{
    "NOERROR":95,
    "FORMERR":0,
    "SERVFAIL":2,
    "NXDOMAIN":3,
    "NOTIMP":0,
    "REFUSED":0,
    "YXDOMAIN":0,
    "YXRRSET":0,
    "NXRRSET":0,
    "NOTAUTH":0,
    "NOTZONE":0,
    "RESERVED11":0,
    "RESERVED12":0,
    "RESERVED13":0,
    "RESERVED14":0,
    "RESERVED15":0,
    "BADVERS":0,
    "17":0,
    "18":0,
    "19":0,
    "20":0,
    "21":0,
    "22":0,
    "BADCOOKIE":0
  },
  "views":{
    "_default":{
      "resolver":{
        "stats":{
          "Queryv6":50,
          "Responsev6":48,
          "ValAttempt":10,
          "ValOk":8,
          "QryRTT10":2,
          "BucketSize":17,
          "ClientCookieOut":2,
          "Priming":1
        },
        "qtypes":{
          "NS":5,
          "DNSKEY":3,
          "A":20,
          "AAAA":15
        },
        "cache":{
          "A":13,
          "NS":1,
          "AAAA":13
        },
        "cachestats":{
          "CacheHits":200,
          "CacheMisses":50,
          "QueryHits":0,
          "QueryMisses":0,
          "DeleteLRU":0,
          "DeleteTTL":0,
          "CoveringNSEC":0,
          "CacheNodes":30,
          "CacheNSECNodes":0,
          "CacheBuckets":16
        },
        "adb":{
          "nentries":1021,
          "entriescnt":26,
          "nnames":1021,
          "namescnt":13
        }
      }
    },
    "_bind":{
      "resolver":{
        "stats":{
          "BucketSize":17
        },
        "qtypes":{},
        "cache":{},
        "cachestats":{
          "CacheHits":0,
          "CacheMisses":0,
          "QueryHits":0,
          "QueryMisses":0,
          "DeleteLRU":0,
          "DeleteTTL":0,
          "CoveringNSEC":0,
          "CacheNodes":0,
          "CacheNSECNodes":0,
          "CacheBuckets":16
        },
        "adb":{
          "nentries":1021,
          "nnames":1021
        }
      }
    }
  }
}`))
		case "/json/v1/mem":
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "boot-time":"2026-04-12T16:36:58.355Z",
  "config-time":"2026-04-12T16:36:58.385Z",
  "current-time":"2026-04-12T17:48:06.950Z",
  "version":"9.18.39",
  "memory":{
    "TotalUse":55379219,
    "InUse":43399359,
    "Malloced":43559263,
    "ContextSize":159904,
    "Lost":0,
    "contexts":[
      {
        "id":"0x56125565b9a0",
        "name":"main",
        "references":8790,
        "total":48545588,
        "inuse":42112185
      },
      {
        "id":"0x561255786ed0",
        "name":"zonemgr-pool",
        "references":187,
        "total":2788126,
        "inuse":147580
      }
    ]
  }
}`))
		case "/json/v1/net":
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "boot-time":"2026-04-12T16:36:58.355Z",
  "config-time":"2026-04-12T16:36:58.385Z",
  "current-time":"2026-04-12T19:08:35.911Z",
  "version":"9.18.39",
  "sockstats":{
    "UDP4Open":42,
    "UDP6Open":58,
    "UDP4OpenFail":1,
    "TCP4Open":32,
    "TCP6Open":40,
    "UDP4Close":26,
    "UDP6Close":26,
    "TCP4Close":12,
    "UDP4ConnFail":26,
    "UDP6ConnFail":26,
    "UDP4Conn":10,
    "TCP4Accept":16,
    "UDP4Active":18,
    "UDP6Active":36,
    "TCP4Active":40,
    "TCP6Active":45
  }
}`))
		case "/json/v1/traffic":
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "boot-time":"2026-04-12T19:00:54.866Z",
  "config-time":"2026-04-12T19:00:54.933Z",
  "current-time":"2026-04-12T20:00:45.906Z",
  "version":"9.18.39",
  "traffic":{
    "dns-udp-requests-sizes-received-ipv4":{
      "32-47":10
    },
    "dns-udp-responses-sizes-sent-ipv4":{
      "32-47":1,
      "48-63":2,
      "64-79":1,
      "80-95":6
    },
    "dns-tcp-requests-sizes-received-ipv4":{},
    "dns-tcp-responses-sizes-sent-ipv4":{},
    "dns-udp-requests-sizes-received-ipv6":{},
    "dns-udp-responses-sizes-sent-ipv6":{},
    "dns-tcp-requests-sizes-received-ipv6":{},
    "dns-tcp-responses-sizes-sent-ipv6":{}
  }
}`))
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
}
