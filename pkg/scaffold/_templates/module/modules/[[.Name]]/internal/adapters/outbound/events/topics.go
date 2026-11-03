package events

// domainTopics maps semantic domain event types to Watermill routing keys:
// the single place where domain semantics become transport concerns. Each
// slice registers its events from its own <entity>_events.go; the keys come
// from the (options.v1.topic) option of the proto event messages.
var domainTopics = func() map[string]string {
	m := make(map[string]string)
	// mmw:slice-topic
	return m
}()
