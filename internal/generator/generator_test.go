package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("WaitForFile", func() {
	It("blocks until the file exists", func() {
		path := filepath.Join(GinkgoT().TempDir(), "start")
		released := make(chan error, 1)
		go func() { released <- WaitForFile(context.Background(), path, 5*time.Millisecond) }()

		Consistently(released, 100*time.Millisecond).ShouldNot(Receive())
		Expect(os.WriteFile(path, nil, 0o644)).To(Succeed())
		Eventually(released, time.Second).Should(Receive(BeNil()))
	})
	It("returns when the context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- WaitForFile(ctx, "/nonexistent/start", 5*time.Millisecond) }()
		cancel()
		Eventually(done, time.Second).Should(Receive(MatchError(context.Canceled)))
	})
})

var _ = Describe("bounded emission", func() {
	var (
		g     *LogGenerator
		mu    sync.Mutex
		lines []string
	)
	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), lines...)
	}

	BeforeEach(func() {
		lines = nil
		src, err := NewPayloadSource(&PayloadConfig{Templates: []PayloadTemplate{{Weight: 1, Message: "seq={{SEQ}} id={{LOADER_ID}}"}}}, "9")
		Expect(err).NotTo(HaveOccurred())
		g = &LogGenerator{
			rate:     5,
			payload:  src,
			logCount: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_total"}),
			opts:     Options{LogType: "config", LogFormat: "raw"},
			writeToDestination: func(_, line string, _ LabelSetOptions) error {
				mu.Lock()
				defer mu.Unlock()
				lines = append(lines, line)
				return nil
			},
			deferClose: func() {},
		}
	})

	It("emit stops at the deadline and returns the count", func() {
		n := g.emit(context.Background(), "h", "h", time.Now().Add(250*time.Millisecond))
		Expect(n).To(Equal(int64(5)))
		got := snapshot()
		Expect(got).To(HaveLen(5))
		Expect(got[0]).To(Equal("seq=0 id=9\n"))
		Expect(got[4]).To(Equal("seq=4 id=9\n"))
	})

	It("writes the sent count then idles until cancelled", func() {
		sent := filepath.Join(GinkgoT().TempDir(), "sent")
		g.opts.Duration = 250 * time.Millisecond
		g.opts.SentCountFile = sent

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			g.generateLogs(ctx)
		}()

		Eventually(func() string {
			b, _ := os.ReadFile(sent)
			return strings.TrimSpace(string(b))
		}, 3*time.Second, 20*time.Millisecond).Should(Equal("5"))
		Expect(snapshot()).To(HaveLen(5))

		Consistently(done, 300*time.Millisecond).ShouldNot(BeClosed())
		Expect(snapshot()).To(HaveLen(5))

		cancel()
		Eventually(done, time.Second).Should(BeClosed())
	})

	It("does not emit before the start signal exists", func() {
		start := filepath.Join(GinkgoT().TempDir(), "start")
		g.opts.StartSignalFile = start

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go g.generateLogs(ctx)

		Consistently(snapshot, 300*time.Millisecond).Should(BeEmpty())
		Expect(os.WriteFile(start, nil, 0o644)).To(Succeed())
		Eventually(snapshot, 2*time.Second).ShouldNot(BeEmpty())
	})
})

var _ = Describe("NewLogGenerator", func() {
	It("requires --payload-config for log type config", func() {
		_, err := NewLogGenerator(Options{LogType: "config"}, prometheus.NewRegistry())
		Expect(err).To(MatchError(ContainSubstring("payload-config")))
	})
})
