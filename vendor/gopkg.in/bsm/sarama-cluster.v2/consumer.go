package cluster

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"
)

// Consumer is a cluster group consumer
type Consumer struct {
	client *Client

	csmr sarama.Consumer
	subs *partitionMap

	consumerID   string
	generationID int32
	groupID      string
	memberID     string

	targetTopics []string
	knownTopics  []string

	dying, dead chan none

	consuming     int32
	errors        chan error
	messages      chan *sarama.ConsumerMessage
	notifications chan *Notification

	commitMu sync.Mutex
}

// NewConsumerFromClient initializes a new consumer from an existing client
func NewConsumerFromClient(client *Client, groupID string, topics []string) (*Consumer, error) {
	csmr, err := sarama.NewConsumerFromClient(client.Client)
	if err != nil {
		return nil, err
	}

	sort.Strings(topics)
	c := &Consumer{
		client:  client,
		csmr:    csmr,
		subs:    newPartitionMap(),
		groupID: groupID,

		targetTopics: topics,

		dying: make(chan none),
		dead:  make(chan none),

		errors:        make(chan error, client.config.ChannelBufferSize),
		messages:      make(chan *sarama.ConsumerMessage),
		notifications: make(chan *Notification, 1),
	}
	if err := c.client.RefreshCoordinator(groupID); err != nil {
		return nil, err
	}

	go c.mainLoop()
	return c, nil
}

// NewConsumer initializes a new consumer
func NewConsumer(addrs []string, groupID string, topics []string, config *Config) (*Consumer, error) {
	client, err := NewClient(addrs, config)
	if err != nil {
		return nil, err
	}

	consumer, err := NewConsumerFromClient(client, groupID, topics)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	consumer.client.own = true
	return consumer, nil
}

// Messages returns the read channel for the messages that are returned by
// the broker.
func (c *Consumer) Messages() <-chan *sarama.ConsumerMessage { return c.messages }

// Errors returns a read channel of errors that occur during offset management, if
// enabled. By default, errors are logged and not returned over this channel. If
// you want to implement any custom error handling, set your config's
// Consumer.Return.Errors setting to true, and read from this channel.
func (c *Consumer) Errors() <-chan error { return c.errors }

// Notifications returns a channel of Notifications that occur during consumer
// rebalancing. Notifications will only be emitted over this channel, if your config's
// Group.Return.Notifications setting to true.
func (c *Consumer) Notifications() <-chan *Notification { return c.notifications }

// HighWaterMarks returns the current high water marks for each topic and partition
// Consistency between partitions is not garanteed since high water marks are updated separately.
func (c *Consumer) HighWaterMarks() map[string]map[int32]int64 { return c.csmr.HighWaterMarks() }

// MarkOffset marks the provided message as processed, alongside a metadata string
// that represents the state of the partition consumer at that point in time. The
// metadata string can be used by another consumer to restore that state, so it
// can resume consumption.
//
// Note: calling MarkOffset does not necessarily commit the offset to the backend
// store immediately for efficiency reasons, and it may never be committed if
// your application crashes. This means that you may end up processing the same
// message twice, and your processing should ideally be idempotent.
func (c *Consumer) MarkOffset(msg *sarama.ConsumerMessage, metadata string) {
	c.subs.Fetch(msg.Topic, msg.Partition).MarkOffset(msg.Offset+1, metadata)
}

// MarkPartitionOffset marks an offset of the provided topic/partition as processed.
// See MarkOffset for additional explanation.
func (c *Consumer) MarkPartitionOffset(topic string, partition int32, offset int64, metadata string) {
	c.subs.Fetch(topic, partition).MarkOffset(offset+1, metadata)
}

// Subscriptions returns the consumed topics and partitions
func (c *Consumer) Subscriptions() map[string][]int32 {
	return c.subs.Info()
}

// CommitOffsets manually commits marked offsets
func (c *Consumer) CommitOffsets() error {
	c.commitMu.Lock()
	defer c.commitMu.Unlock()

	req := &sarama.OffsetCommitRequest{
		Version:                 2,
		ConsumerGroup:           c.groupID,
		ConsumerGroupGeneration: c.generationID,
		ConsumerID:              c.memberID,
		RetentionTime:           -1,
	}

	if rt := c.client.config.Consumer.Offsets.Retention; rt != 0 {
		req.RetentionTime = int64(rt / time.Millisecond)
	}

	var dirty bool
	snap := c.subs.Snapshot()
	for tp, state := range snap {
		if state.Dirty {
			req.AddBlock(tp.Topic, tp.Partition, state.Info.Offset, 0, state.Info.Metadata)
			dirty = true
		}
	}
	if !dirty {
		return nil
	}

	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return err
	}

	resp, err := broker.CommitOffset(req)
	if err != nil {
		c.closeCoordinator(broker, err)
		return err
	}

	for topic, perrs := range resp.Errors {
		for partition, kerr := range perrs {
			if kerr != sarama.ErrNoError {
				err = kerr
			} else if state, ok := snap[topicPartition{topic, partition}]; ok {
				c.subs.Fetch(topic, partition).MarkCommitted(state.Info.Offset)
			}
		}
	}

	return err
}

// Close safely closes the consumer and releases all resources
func (c *Consumer) Close() (err error) {
	select {
	case <-c.dying:
		return
	default:
		close(c.dying)
	}
	<-c.dead

	if e := c.release(); e != nil {
		err = e
	}
	if e := c.csmr.Close(); e != nil {
		err = e
	}
	close(c.messages)
	close(c.errors)

	if e := c.leaveGroup(); e != nil {
		err = e
	}
	close(c.notifications)

	if c.client.own {
		if e := c.client.Close(); e != nil {
			err = e
		}
	}

	return
}

func (c *Consumer) mainLoop() {
	defer close(c.dead)
	defer atomic.StoreInt32(&c.consuming, 0)

	for {
		atomic.StoreInt32(&c.consuming, 0)

		// Check if close was requested
		select {
		case <-c.dying:
			return
		default:
		}

		// Remember previous subscriptions
		var notification *Notification
		if c.client.config.Group.Return.Notifications {
			notification = newNotification(c.subs.Info())
		}

		// Rebalance, fetch new subscriptions
		subs, err := c.rebalance()
		if err != nil {
			c.rebalanceError(err, notification)
			continue
		}

		// Start the heartbeat
		hbStop, hbDone := make(chan none), make(chan none)
		go c.hbLoop(hbStop, hbDone)

		// Subscribe to topic/partitions
		if err := c.subscribe(subs); err != nil {
			close(hbStop)
			<-hbDone
			c.rebalanceError(err, notification)
			continue
		}

		// Start topic watcher loop
		twStop, twDone := make(chan none), make(chan none)
		go c.twLoop(twStop, twDone)

		// Start consuming and comitting offsets
		cmStop, cmDone := make(chan none), make(chan none)
		go c.cmLoop(cmStop, cmDone)
		atomic.StoreInt32(&c.consuming, 1)

		// Update notification with new claims
		if c.client.config.Group.Return.Notifications {
			notification.claim(subs)
			c.notifications <- notification
		}

		// Wait for signals
		select {
		case <-hbDone:
			close(cmStop)
			close(twStop)
			<-cmDone
			<-twDone
		case <-twDone:
			close(cmStop)
			close(hbStop)
			<-cmDone
			<-hbDone
		case <-cmDone:
			close(twStop)
			close(hbStop)
			<-twDone
			<-hbDone
		case <-c.dying:
			close(cmStop)
			close(twStop)
			close(hbStop)
			<-cmDone
			<-twDone
			<-hbDone
			return
		}
	}
}

// heartbeat loop, triggered by the mainLoop
func (c *Consumer) hbLoop(stop <-chan none, done chan<- none) {
	defer close(done)

	ticker := time.NewTicker(c.client.config.Group.Heartbeat.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			switch err := c.heartbeat(); err {
			case nil, sarama.ErrNoError:
			case sarama.ErrNotCoordinatorForConsumer, sarama.ErrRebalanceInProgress:
				return
			default:
				c.handleError(&Error{Ctx: "heartbeat", error: err})
				return
			}
		case <-stop:
			return
		}
	}
}

// topic watcher loop, triggered by the mainLoop
func (c *Consumer) twLoop(stop <-chan none, done chan<- none) {
	defer close(done)

	// ticker := time.NewTicker(c.client.config.Metadata.RefreshFrequency)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			topics, err := c.client.Topics()
			if err != nil {
				c.handleError(&Error{Ctx: "topics", error: err})
				return
			}

			for _, topic := range topics {
				if c.isTargetTopic(topic) && !c.isKnownTopic(topic) {
					return
				}
			}
		case <-stop:
			return
		}
	}
}

// commit loop, triggered by the mainLoop
func (c *Consumer) cmLoop(stop <-chan none, done chan<- none) {
	defer close(done)

	ticker := time.NewTicker(c.client.config.Consumer.Offsets.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.commitOffsetsWithRetry(c.client.config.Group.Offsets.Retry.Max); err != nil {
				c.handleError(&Error{Ctx: "commit", error: err})
				return
			}
		case <-stop:
			return
		}
	}
}

func (c *Consumer) rebalanceError(err error, notification *Notification) {
	if c.client.config.Group.Return.Notifications {
		c.notifications <- notification
	}

	switch err {
	case sarama.ErrRebalanceInProgress:
	default:
		c.handleError(&Error{Ctx: "rebalance", error: err})
	}

	select {
	case <-c.dying:
	case <-time.After(c.client.config.Metadata.Retry.Backoff):
	}
}

func (c *Consumer) handleError(e *Error) {
	if c.client.config.Consumer.Return.Errors {
		select {
		case c.errors <- e:
		case <-c.dying:
			return
		}
	} else {
		sarama.Logger.Printf("%s error: %s\n", e.Ctx, e.Error())
	}
}

// Releases the consumer and commits offsets, called from rebalance() and Close()
func (c *Consumer) release() (err error) {
	// Stop all consumers
	c.subs.Stop()

	// Clear subscriptions on exit
	defer c.subs.Clear()

	// Wait for messages to be processed
	time.Sleep(c.client.config.Consumer.MaxProcessingTime)

	// Commit offsets, continue on errors
	if e := c.commitOffsetsWithRetry(c.client.config.Group.Offsets.Retry.Max); e != nil {
		err = e
	}

	return
}

// --------------------------------------------------------------------

// Performs a heartbeat, part of the mainLoop()
func (c *Consumer) heartbeat() error {
	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return err
	}

	resp, err := broker.Heartbeat(&sarama.HeartbeatRequest{
		GroupId:      c.groupID,
		MemberId:     c.memberID,
		GenerationId: c.generationID,
	})
	if err != nil {
		c.closeCoordinator(broker, err)
		return err
	}
	return resp.Err
}

// Performs a rebalance, part of the mainLoop()
func (c *Consumer) rebalance() (map[string][]int32, error) {
	sarama.Logger.Printf("cluster/consumer %s rebalance\n", c.memberID)

	if err := c.client.RefreshCoordinator(c.groupID); err != nil {
		return nil, err
	}

	allTopics, err := c.client.Topics()
	if err != nil {
		return nil, err
	}
	c.knownTopics = c.onlyConsumableTopics(allTopics)
	sort.Strings(c.knownTopics)

	// Release subscriptions
	if err := c.release(); err != nil {
		return nil, err
	}

	// Re-join consumer group
	strategy, err := c.joinGroup()
	switch {
	case err == sarama.ErrUnknownMemberId:
		c.memberID = ""
		return nil, err
	case err != nil:
		return nil, err
	}
	// sarama.Logger.Printf("cluster/consumer %s/%d joined group %s\n", c.memberID, c.generationID, c.groupID)

	// Sync consumer group state, fetch subscriptions
	subs, err := c.syncGroup(strategy)
	switch {
	case err == sarama.ErrRebalanceInProgress:
		return nil, err
	case err != nil:
		_ = c.leaveGroup()
		return nil, err
	}
	return subs, nil
}

// Performs the subscription, part of the mainLoop()
func (c *Consumer) subscribe(subs map[string][]int32) error {
	// fetch offsets
	offsets, err := c.fetchOffsets(subs)
	if err != nil {
		_ = c.leaveGroup()
		return err
	}

	// create consumers in parallel
	var mu sync.Mutex
	var wg sync.WaitGroup

	for topic, partitions := range subs {
		for _, partition := range partitions {
			wg.Add(1)

			info := offsets[topic][partition]
			go func(t string, p int32) {
				if e := c.createConsumer(t, p, info); e != nil {
					mu.Lock()
					err = e
					mu.Unlock()
				}
				wg.Done()
			}(topic, partition)
		}
	}
	wg.Wait()

	if err != nil {
		_ = c.release()
		_ = c.leaveGroup()
	}
	return err
}

// --------------------------------------------------------------------

// Send a request to the broker to join group on rebalance()
func (c *Consumer) joinGroup() (*balancer, error) {
	req := &sarama.JoinGroupRequest{
		GroupId:        c.groupID,
		MemberId:       c.memberID,
		SessionTimeout: int32(c.client.config.Group.Session.Timeout / time.Millisecond),
		ProtocolType:   "consumer",
	}

	meta := &sarama.ConsumerGroupMemberMetadata{
		Version: 1,
		Topics:  c.knownTopics,
	}
	err := req.AddGroupProtocolMetadata(string(StrategyRange), meta)
	if err != nil {
		return nil, err
	}
	err = req.AddGroupProtocolMetadata(string(StrategyRoundRobin), meta)
	if err != nil {
		return nil, err
	}

	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	}

	resp, err := broker.JoinGroup(req)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	} else if resp.Err != sarama.ErrNoError {
		c.closeCoordinator(broker, resp.Err)
		return nil, resp.Err
	}

	var strategy *balancer
	if resp.LeaderId == resp.MemberId {
		members, err := resp.GetMembers()
		if err != nil {
			return nil, err
		}

		strategy, err = newBalancerFromMeta(c.client, members)
		if err != nil {
			return nil, err
		}
	}

	c.memberID = resp.MemberId
	c.generationID = resp.GenerationId

	return strategy, nil
}

// Send a request to the broker to sync the group on rebalance().
// Returns a list of topics and partitions to consume.
func (c *Consumer) syncGroup(strategy *balancer) (map[string][]int32, error) {
	req := &sarama.SyncGroupRequest{
		GroupId:      c.groupID,
		MemberId:     c.memberID,
		GenerationId: c.generationID,
	}

	for memberID, topics := range strategy.Perform(c.client.config.Group.PartitionStrategy) {
		if err := req.AddGroupAssignmentMember(memberID, &sarama.ConsumerGroupMemberAssignment{
			Version: 1,
			Topics:  topics,
		}); err != nil {
			return nil, err
		}
	}

	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	}

	resp, err := broker.SyncGroup(req)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	} else if resp.Err != sarama.ErrNoError {
		c.closeCoordinator(broker, resp.Err)
		return nil, resp.Err
	}

	// Return if there is nothing to subscribe to
	if len(resp.MemberAssignment) == 0 {
		return nil, nil
	}

	// Get assigned subscriptions
	members, err := resp.GetMemberAssignment()
	if err != nil {
		return nil, err
	}

	// Sort partitions, for each topic
	for topic := range members.Topics {
		sort.Sort(int32Slice(members.Topics[topic]))
	}
	return members.Topics, nil
}

// Fetches latest committed offsets for all subscriptions
func (c *Consumer) fetchOffsets(subs map[string][]int32) (map[string]map[int32]offsetInfo, error) {
	offsets := make(map[string]map[int32]offsetInfo, len(subs))
	req := &sarama.OffsetFetchRequest{
		Version:       1,
		ConsumerGroup: c.groupID,
	}

	for topic, partitions := range subs {
		offsets[topic] = make(map[int32]offsetInfo, len(partitions))
		for _, partition := range partitions {
			offsets[topic][partition] = offsetInfo{Offset: -1}
			req.AddPartition(topic, partition)
		}
	}

	// Wait for other cluster consumers to process, release and commit
	time.Sleep(c.client.config.Consumer.MaxProcessingTime * 2)

	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	}

	resp, err := broker.FetchOffset(req)
	if err != nil {
		c.closeCoordinator(broker, err)
		return nil, err
	}

	for topic, partitions := range subs {
		for _, partition := range partitions {
			block := resp.GetBlock(topic, partition)
			if block == nil {
				return nil, sarama.ErrIncompleteResponse
			}

			if block.Err == sarama.ErrNoError {
				offsets[topic][partition] = offsetInfo{Offset: block.Offset, Metadata: block.Metadata}
			} else {
				return nil, block.Err
			}
		}
	}
	return offsets, nil
}

// Send a request to the broker to leave the group on failes rebalance() and on Close()
func (c *Consumer) leaveGroup() error {
	broker, err := c.client.Coordinator(c.groupID)
	if err != nil {
		c.closeCoordinator(broker, err)
		return err
	}

	if _, err = broker.LeaveGroup(&sarama.LeaveGroupRequest{
		GroupId:  c.groupID,
		MemberId: c.memberID,
	}); err != nil {
		c.closeCoordinator(broker, err)
	}
	return err
}

// --------------------------------------------------------------------

func (c *Consumer) createConsumer(topic string, partition int32, info offsetInfo) error {
	sarama.Logger.Printf("cluster/consumer %s consume %s/%d from %d\n", c.memberID, topic, partition, info.NextOffset(c.client.config.Consumer.Offsets.Initial))

	// Create partitionConsumer
	pc, err := newPartitionConsumer(c.csmr, topic, partition, info, c.client.config.Consumer.Offsets.Initial)
	if err != nil {
		return err
	}

	// Store in subscriptions
	c.subs.Store(topic, partition, pc)

	// Start partition consumer goroutine
	go pc.Loop(c.messages, c.errors)

	return nil
}

func (c *Consumer) commitOffsetsWithRetry(retries int) error {
	err := c.CommitOffsets()
	if err != nil && retries > 0 && c.subs.HasDirty() {
		return c.commitOffsetsWithRetry(retries - 1)
	}
	return err
}

func (c *Consumer) closeCoordinator(broker *sarama.Broker, err error) {
	if broker != nil {
		_ = broker.Close()
	}

	switch err {
	case sarama.ErrConsumerCoordinatorNotAvailable, sarama.ErrNotCoordinatorForConsumer:
		_ = c.client.RefreshCoordinator(c.groupID)
	}
}

func (c *Consumer) onlyConsumableTopics(allTopics []string) []string {
	topics := allTopics[:0]
	for _, topic := range allTopics {
		if c.isTargetTopic(topic) {
			topics = append(topics, topic)
		}
	}
	return topics
}

func (c *Consumer) isTargetTopic(topic string) bool {
	pos := sort.SearchStrings(c.targetTopics, topic)
	return pos < len(c.targetTopics) && c.targetTopics[pos] == topic
}

func (c *Consumer) isKnownTopic(topic string) bool {
	pos := sort.SearchStrings(c.knownTopics, topic)
	return pos < len(c.knownTopics) && c.knownTopics[pos] == topic
}
