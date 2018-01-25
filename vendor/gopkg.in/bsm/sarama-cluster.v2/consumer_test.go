package cluster

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Consumer", func() {

	var newConsumer = func(group string) (*Consumer, error) {
		config := NewConfig()
		config.Consumer.Return.Errors = true
		return NewConsumer(testKafkaAddrs, group, testTopics, config)
	}

	var newConsumerOf = func(group string, topics ...string) (*Consumer, error) {
		config := NewConfig()
		config.Consumer.Return.Errors = true
		return NewConsumer(testKafkaAddrs, group, topics, config)
	}

	var subscriptionsOf = func(c *Consumer) GomegaAsyncAssertion {
		return Eventually(func() map[string][]int32 {
			return c.Subscriptions()
		}, "10s", "100ms")
	}

	var consume = func(consumerID, group string, max int, out chan *testConsumerMessage) {
		go func() {
			defer GinkgoRecover()

			cs, err := newConsumer(group)
			Expect(err).NotTo(HaveOccurred())
			defer cs.Close()
			cs.consumerID = consumerID

			for msg := range cs.Messages() {
				out <- &testConsumerMessage{*msg, consumerID}
				cs.MarkOffset(msg, "")

				if max--; max == 0 {
					return
				}
			}
		}()
	}

	It("should init and share", func() {
		// start CS1
		cs1, err := newConsumer(testGroup)
		Expect(err).NotTo(HaveOccurred())

		// CS1 should consume all 8 partitions
		subscriptionsOf(cs1).Should(Equal(map[string][]int32{
			"topic-a": {0, 1, 2, 3},
			"topic-b": {0, 1, 2, 3},
		}))

		// start CS2
		cs2, err := newConsumer(testGroup)
		Expect(err).NotTo(HaveOccurred())
		defer cs2.Close()

		// CS1 and CS2 should consume 4 partitions each
		subscriptionsOf(cs1).Should(HaveLen(2))
		subscriptionsOf(cs1).Should(HaveKeyWithValue("topic-a", HaveLen(2)))
		subscriptionsOf(cs1).Should(HaveKeyWithValue("topic-b", HaveLen(2)))

		subscriptionsOf(cs2).Should(HaveLen(2))
		subscriptionsOf(cs2).Should(HaveKeyWithValue("topic-a", HaveLen(2)))
		subscriptionsOf(cs2).Should(HaveKeyWithValue("topic-b", HaveLen(2)))

		// shutdown CS1, now CS2 should consume all 8 partitions
		Expect(cs1.Close()).NotTo(HaveOccurred())
		subscriptionsOf(cs2).Should(Equal(map[string][]int32{
			"topic-a": {0, 1, 2, 3},
			"topic-b": {0, 1, 2, 3},
		}))
	})

	It("should allow more consumers than partitions", func() {
		cs1, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())
		defer cs1.Close()
		cs2, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())
		defer cs2.Close()
		cs3, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())
		defer cs3.Close()
		cs4, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())

		// start 4 consumers, one for each partition
		subscriptionsOf(cs1).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs2).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs3).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs4).Should(HaveKeyWithValue("topic-a", HaveLen(1)))

		// add a 5th consumer
		cs5, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())
		defer cs5.Close()

		// make sure no errors occurred
		Expect(cs1.Errors()).ShouldNot(Receive())
		Expect(cs2.Errors()).ShouldNot(Receive())
		Expect(cs3.Errors()).ShouldNot(Receive())
		Expect(cs4.Errors()).ShouldNot(Receive())
		Expect(cs5.Errors()).ShouldNot(Receive())

		// close 4th, make sure the 5th takes over
		Expect(cs4.Close()).To(Succeed())
		subscriptionsOf(cs1).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs2).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs3).Should(HaveKeyWithValue("topic-a", HaveLen(1)))
		subscriptionsOf(cs4).Should(BeEmpty())
		subscriptionsOf(cs5).Should(HaveKeyWithValue("topic-a", HaveLen(1)))

		// there should still be no errors
		Expect(cs1.Errors()).ShouldNot(Receive())
		Expect(cs2.Errors()).ShouldNot(Receive())
		Expect(cs3.Errors()).ShouldNot(Receive())
		Expect(cs4.Errors()).ShouldNot(Receive())
		Expect(cs5.Errors()).ShouldNot(Receive())
	})

	It("should be allowed to subscribe to partitions that do not exist (yet)", func() {
		cs, err := newConsumerOf(testGroup, append([]string{"topic-c"}, testTopics...)...)
		Expect(err).NotTo(HaveOccurred())
		defer cs.Close()
		subscriptionsOf(cs).Should(Equal(map[string][]int32{
			"topic-a": {0, 1, 2, 3},
			"topic-b": {0, 1, 2, 3},
		}))
	})

	It("should support manual mark/commit", func() {
		cs, err := newConsumerOf(testGroup, "topic-a")
		Expect(err).NotTo(HaveOccurred())
		defer cs.Close()

		subscriptionsOf(cs).Should(Equal(map[string][]int32{
			"topic-a": {0, 1, 2, 3}},
		))

		cs.MarkPartitionOffset("topic-a", 1, 3, "")
		cs.MarkPartitionOffset("topic-a", 2, 4, "")
		Expect(cs.CommitOffsets()).NotTo(HaveOccurred())

		offsets, err := cs.fetchOffsets(cs.Subscriptions())
		Expect(err).NotTo(HaveOccurred())
		Expect(offsets).To(Equal(map[string]map[int32]offsetInfo{
			"topic-a": {0: {Offset: -1}, 1: {Offset: 4}, 2: {Offset: 5}, 3: {Offset: -1}},
		}))
	})

	It("should consume/commit/resume", func() {
		acc := make(chan *testConsumerMessage, 150000)
		consume("A", "fuzzing", 1500, acc)
		consume("B", "fuzzing", 2000, acc)
		consume("C", "fuzzing", 1500, acc)
		consume("D", "fuzzing", 200, acc)
		consume("E", "fuzzing", 100, acc)

		Expect(testSeed(5000)).NotTo(HaveOccurred())
		Eventually(func() int { return len(acc) }, "30s", "100ms").Should(BeNumerically(">=", 5000))

		consume("F", "fuzzing", 300, acc)
		consume("G", "fuzzing", 400, acc)
		consume("H", "fuzzing", 1000, acc)
		consume("I", "fuzzing", 2000, acc)
		Expect(testSeed(5000)).NotTo(HaveOccurred())
		Eventually(func() int { return len(acc) }, "30s", "100ms").Should(BeNumerically(">=", 8000))

		consume("J", "fuzzing", 1000, acc)
		Expect(testSeed(5000)).NotTo(HaveOccurred())
		Eventually(func() int { return len(acc) }, "30s", "100ms").Should(BeNumerically(">=", 9000))

		consume("K", "fuzzing", 1000, acc)
		consume("L", "fuzzing", 3000, acc)
		Expect(testSeed(5000)).NotTo(HaveOccurred())
		Eventually(func() int { return len(acc) }, "30s", "100ms").Should(BeNumerically(">=", 12000))

		consume("M", "fuzzing", 1000, acc)
		Expect(testSeed(5000)).NotTo(HaveOccurred())
		Eventually(func() int { return len(acc) }, "30s", "100ms").Should(BeNumerically(">=", 15000))

		close(acc)

		uniques := make(map[string][]string)
		for msg := range acc {
			key := fmt.Sprintf("%s/%d/%d", msg.Topic, msg.Partition, msg.Offset)
			uniques[key] = append(uniques[key], msg.ConsumerID)
		}
		Expect(uniques).To(HaveLen(15000))
	})

	It("should allow close to be called multiple times", func() {
		cs, err := newConsumer(testGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(cs.Close()).NotTo(HaveOccurred())
		Expect(cs.Close()).NotTo(HaveOccurred())
	})

})
