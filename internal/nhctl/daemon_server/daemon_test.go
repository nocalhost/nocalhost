package daemon_server

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
	"testing"
)

func TestUnMar(t *testing.T) {
	meta := &appmeta.ApplicationMeta{}

	meta.Config = &profile2.NocalHostAppConfigV2{}
	meta.Config.ApplicationConfig = &profile2.ApplicationConfig{}

	meta.Config.ApplicationConfig.HelmVals = map[string]interface{}{
		"service": map[interface{}]interface{}{"port": "9082"},
		"bookinfo": map[string]interface{}{
			"deploy": map[string]interface{}{
				"resources": map[interface{}]interface{}{"limit": map[interface{}]interface{}{"cpu": "700m"}},
			},
		},
	}

	if m, err := yaml.Marshal(meta); err != nil {
		t.Error(err)
	} else {
		println(string(m))

		metas := &appmeta.ApplicationMeta{}
		_ = yaml.Unmarshal(m, metas)

		if massss, err := json.Marshal(metas); err != nil {
			t.Error(err)
		}else {
			println(string(massss))
		}
	}
}
