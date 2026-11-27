package generator

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func writeConfig(content string) string {
	path := filepath.Join(GinkgoT().TempDir(), "payload.json")
	Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
	return path
}

var _ = Describe("LoadPayloadConfig", func() {
	It("loads a valid config", func() {
		cfg, err := LoadPayloadConfig(writeConfig(`{"seed":7,"templates":[{"weight":1,"message":"a"},{"weight":0,"message":"b"}]}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Seed).To(Equal(7))
		Expect(cfg.Templates).To(HaveLen(2))
	})
	It("fails on a missing file", func() {
		_, err := LoadPayloadConfig(filepath.Join(GinkgoT().TempDir(), "nope.json"))
		Expect(err).To(HaveOccurred())
	})
	It("fails on bad JSON", func() {
		_, err := LoadPayloadConfig(writeConfig(`{not json`))
		Expect(err).To(MatchError(ContainSubstring("parse")))
	})
	It("fails with no templates", func() {
		_, err := LoadPayloadConfig(writeConfig(`{"seed":1,"templates":[]}`))
		Expect(err).To(MatchError(ContainSubstring("at least one template")))
	})
	It("fails on a negative weight", func() {
		_, err := LoadPayloadConfig(writeConfig(`{"templates":[{"weight":-1,"message":"a"},{"weight":5,"message":"b"}]}`))
		Expect(err).To(MatchError(ContainSubstring("negative")))
	})
	It("fails when all weights are zero", func() {
		_, err := LoadPayloadConfig(writeConfig(`{"templates":[{"weight":0,"message":"a"}]}`))
		Expect(err).To(MatchError(ContainSubstring("greater than zero")))
	})
})

var _ = Describe("PayloadSource", func() {
	cfg := func() *PayloadConfig {
		return &PayloadConfig{Seed: 42, Templates: []PayloadTemplate{
			{Weight: 3, Message: "level=info seq={{SEQ}} loader={{LOADER_ID}}"},
			{Weight: 1, Message: "level=error seq={{SEQ}} loader={{LOADER_ID}}"},
		}}
	}
	gen := func(c *PayloadConfig, id string, n int) []string {
		s, err := NewPayloadSource(c, id)
		Expect(err).NotTo(HaveOccurred())
		out := make([]string, n)
		for i := range out {
			out[i] = s.Next()
		}
		return out
	}

	It("is deterministic for a fixed seed", func() {
		Expect(gen(cfg(), "x", 200)).To(Equal(gen(cfg(), "x", 200)))
	})
	It("follows the weights approximately", func() {
		infos := 0
		const n = 20000
		for _, m := range gen(cfg(), "x", n) {
			if strings.HasPrefix(m, "level=info") {
				infos++
			}
		}
		Expect(float64(infos) / n).To(BeNumerically("~", 0.75, 0.03))
	})
	It("never selects zero-weight templates", func() {
		c := cfg()
		c.Templates[0].Weight = 0
		for _, m := range gen(c, "x", 500) {
			Expect(m).To(HavePrefix("level=error"))
		}
	})
	It("increments SEQ from 0 and substitutes LOADER_ID", func() {
		for i, m := range gen(cfg(), "3", 50) {
			Expect(m).To(ContainSubstring("seq=%d ", i))
			Expect(m).To(HaveSuffix("loader=3"))
			Expect(m).NotTo(ContainSubstring("{{"))
		}
	})
	It("substitutes an empty LOADER_ID by default", func() {
		Expect(gen(cfg(), "", 1)[0]).To(HaveSuffix("loader="))
	})
})
