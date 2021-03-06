package collector_test

import (
	"sync"

	"code.cloudfoundry.org/lager"
	. "github.com/cloudfoundry/streaming-mysql-backup-tool/collector"
	"github.com/cloudfoundry/streaming-mysql-backup-tool/collector/collectorfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collector", func() {
	var (
		fakeExecutor *collectorfakes.FakeScriptExecutor
		testLogger   lager.Logger
		collector    *Collector
		wg           *sync.WaitGroup
	)

	BeforeEach(func() {
		fakeExecutor = new(collectorfakes.FakeScriptExecutor)
		testLogger = lager.NewLogger("collector-test")
		collector = NewCollector(fakeExecutor, testLogger)

		wg = &sync.WaitGroup{}

		fakeExecutor.ExecuteStub = func() error {
			return nil
		}
	})

	It("tells the executor to repeatedly execute until stop is called", func() {
		wg.Add(1)
		go func() {
			collector.Start()
			wg.Done()
		}()
		Eventually(fakeExecutor.ExecuteCallCount).Should(BeNumerically(">", 1))

		collector.Stop()
		wg.Wait()

		postStopExecuteCallCount := fakeExecutor.ExecuteCallCount()
		Consistently(fakeExecutor.ExecuteCallCount()).Should(Equal(postStopExecuteCallCount))
	})
})
