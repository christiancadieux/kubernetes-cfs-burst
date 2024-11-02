package burst

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"os"
	"strconv"
)

const (
	WATCH_TIMEOUT      = int64(30 * 60)
	CFS_BURST_FILE     = "cpu.cfs_burst_us"
	CFS_QUOTA_FILE     = "cpu.cfs_quota_us"
	RDEI_BURST_PERCENT = "cfs.io/burst_percent"
)

type BurstManager struct {
	logger       *logrus.Logger
	ctx          context.Context
	nodeName     string
	clientset    *kubernetes.Clientset
	namespaceMgr *NamespaceMgr
	cgroupPath   string
	dryRun       bool
}

// NewBurstManager - monitor pods, find containers and update burst value.
func NewBurstManager(logger *logrus.Logger, ctx context.Context, dryRun bool, clientset *kubernetes.Clientset, nodeName string,
	namespaceMgr *NamespaceMgr, cgroupPath string) *BurstManager {

	return &BurstManager{logger, ctx, nodeName, clientset, namespaceMgr, cgroupPath, dryRun}
}

// Run - watch pods and update their container cgroup info
// New pods generate an 'Added' event followed by 'Modified' events.
// We monitor both because the 'Added' event is generated before all the cgroup directories are present.
func (b *BurstManager) Run() error {

	timeOut := WATCH_TIMEOUT
	podI := b.clientset.CoreV1().Pods("")

	watcher, err := podI.Watch(b.ctx, metav1.ListOptions{
		TimeoutSeconds: &timeOut,
		FieldSelector:  "spec.nodeName=" + b.nodeName,
	})
	if err != nil {
		return err
	}
	for {
		event := <-watcher.ResultChan()
		if event.Type == "" { // reset Watch
			watcher, err = podI.Watch(b.ctx, metav1.ListOptions{
				TimeoutSeconds: &timeOut,
				FieldSelector:  "spec.nodeName=" + b.nodeName,
			})
			if err != nil {
				return err
			}
			continue
		}
		item := event.Object.(*corev1.Pod)
		b.logger.Infof("Reading pod event %s - %s", item.Name, event.Type)
		switch event.Type {
		case watch.Error:
			b.logger.Errorf("Event ERROR - %v", item)
		case watch.Modified:
			err = b.configurePod(item)
			if err != nil {
				b.logger.Error(err)
			}
		case watch.Added:
			err = b.configurePod(item)
			if err != nil {
				b.logger.Error(err)
			}
		}
	}

}

func (b *BurstManager) configurePod(item *corev1.Pod) error {
	burstPercent := b.namespaceMgr.GetBurstPercent(item.Namespace)
	if burstPercent == 0 {
		b.logger.Debugf("burst percent=0 - %s", item.Namespace)
		return nil
	}
	b.logger.Debugf("burst percent for %s = %d", item.Namespace, burstPercent)

	dirFilter := b.cgroupPath + "/burstable/pod" + string(item.UID)
	entries, err := os.ReadDir(dirFilter)
	if err != nil {
		return err
	}
	// for every sub-directory that includes a cfs-quota value, update the burst-value
	for _, dirname := range entries {
		if !dirname.IsDir() {
			continue
		}
		filename := dirFilter + "/" + dirname.Name() + "/" + CFS_QUOTA_FILE
		quotaValue, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		quota, err := readInt(quotaValue)
		if err != nil {
			return fmt.Errorf("ParseInt - %v", err)
		}
		if quota <= 0 {
			continue
		}
		calcBurst := int(quota) * burstPercent / 100

		burstFile := filename[0:len(filename)-len(CFS_QUOTA_FILE)] + CFS_BURST_FILE
		currBurstValue, err := os.ReadFile(burstFile)
		if err != nil {
			return fmt.Errorf("ReadFile - %v", err)
		}
		currBurst, err := readInt(currBurstValue)
		if err != nil {
			return fmt.Errorf("ParseInt - %v", err)
		}
		b.logger.Debugf("Container file=", burstFile, ", quota=", quota, ", burst%=", burstPercent, ", current-burst=", currBurst, ", calc-burst=", calcBurst)
		if int(currBurst) == calcBurst { // nothing to do
			continue
		}
		b.logger.Debugf("  - Update %s to  %d \n", CFS_BURST_FILE, calcBurst)
		if !b.dryRun {
			os.WriteFile(burstFile, []byte(fmt.Sprintf("%d", calcBurst)), 0644)
		}
	}

	return nil
}

func readInt(v []byte) (int64, error) {
	return strconv.ParseInt(string(v[:len(v)-1]), 10, 64)
}
