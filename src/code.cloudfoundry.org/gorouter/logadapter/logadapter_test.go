package logadapter

import (
	"log/slog"
	"strings"

	"code.cloudfoundry.org/gorouter/test_util"
	"code.cloudfoundry.org/lager/v3"
	"github.com/onsi/gomega/gbytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zapLevelSink using testSink", func() {
	var (
		testSink *test_util.TestSink
		logger   *slog.Logger
		sink     *zapLevelSink
	)

	BeforeEach(func() {
		testSink = &test_util.TestSink{Buffer: gbytes.NewBuffer()}
		handler := slog.NewTextHandler(testSink.Buffer, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger = slog.New(handler)
		sink = NewZapLevelSink(logger)
	})

	Describe("NewZapLevelSink", func() {
		It("should return a non-nil zapLevelSink", func() {
			Expect(sink).ToNot(BeNil())
			Expect(sink).To(BeAssignableToTypeOf(&zapLevelSink{}))
		})

		It("should embed the provided logger", func() {
			Expect(sink.logger).To(Equal(logger))
			Expect(sink.logger).To(BeAssignableToTypeOf(&slog.Logger{}))
		})
	})

	Describe("SetMinLevel and LogLevel", func() {
		It("should correctly store and retrieve min log level", func() {
			lagerLogLevel, err := lager.LogLevelFromString(strings.ToLower(slog.LevelError.String()))
			Expect(err).ToNot(HaveOccurred())
			sink.SetMinLevel(lagerLogLevel)
			Expect(sink.LogLevel()).To(Equal(lager.ERROR))
		})
	})
})
