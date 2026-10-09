<<<<<<<< HEAD:core/podcast/podcast_suite_test.go
package podcast_test
========
package netguard_test
>>>>>>>> origin/master:utils/netguard/netguard_suite_test.go

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

<<<<<<<< HEAD:core/podcast/podcast_suite_test.go
func TestPodcast(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Podcast Suite")
========
func TestNetguard(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netguard Suite")
>>>>>>>> origin/master:utils/netguard/netguard_suite_test.go
}
