package burst

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"sync"
)

type NamespaceMgr struct {
	logger                *logrus.Logger
	ctx                   context.Context
	clientset             *kubernetes.Clientset
	nodeName              string
	namespaceBurstPercent map[string]int
	maxBurstPercent       int
	sync.Mutex
}

// NewNamespaceManager - monitor new/updated namespaces and save burst% in namespaceBurstPercent
func NewNamespaceManager(logger *logrus.Logger, ctx context.Context, clientset *kubernetes.Clientset, nodeName string, maxBurstPercent int) *NamespaceMgr {
	return &NamespaceMgr{
		logger:                logger,
		ctx:                   ctx,
		clientset:             clientset,
		nodeName:              nodeName,
		namespaceBurstPercent: map[string]int{},
		maxBurstPercent:       maxBurstPercent,
	}
}

func (ns *NamespaceMgr) Watch() error {

	timeOut := WATCH_TIMEOUT // close the watch every 30 minutes
	podI := ns.clientset.CoreV1().Namespaces()
	watcher, err := podI.Watch(ns.ctx, metav1.ListOptions{TimeoutSeconds: &timeOut})
	if err != nil {
		return err
	}

	for {
		event := <-watcher.ResultChan()
		if event.Type == "" { // get a new watch
			watcher, err = podI.Watch(ns.ctx, metav1.ListOptions{TimeoutSeconds: &timeOut})
			if err != nil {
				return err
			}
			continue
		}
		item := event.Object.(*corev1.Namespace)
		ns.logger.Debugf("Reading namespace event %s - %s", item.Name, event.Type)
		switch event.Type {
		case watch.Error:
			fmt.Println("Event ERROR", item)
		case watch.Modified:
			ns.updateNs(item)
		case watch.Deleted:
			ns.deleteNs(item)
		case watch.Added:
			ns.updateNs(item)
		}
		ns.Print()
	}
}

// GetBurstPercent - get current burst% for a namespace
func (ns *NamespaceMgr) GetBurstPercent(namespace string) int {
	ns.Lock()
	defer ns.Unlock()
	if s1, ok := ns.namespaceBurstPercent[namespace]; ok {
		return s1
	}
	return 0
}

func (ns *NamespaceMgr) Print() {
	ns.Lock()
	defer ns.Unlock()
	if len(ns.namespaceBurstPercent) > 0 {
		ns.logger.Infof("NS-MAP: %+v \n", ns.namespaceBurstPercent)
	}
}

func (ns *NamespaceMgr) updateNs(item *corev1.Namespace) {
	ns.Lock()
	defer ns.Unlock()
	if item.Annotations != nil {
		if s1, ok := item.Annotations[RDEI_BURST_PERCENT]; ok {
			val, err := strconv.Atoi(s1)
			if err == nil {
				if val < 0 || val > ns.maxBurstPercent {
					ns.logger.Warningf("%s - Percentage %d is invalid (max=%d)", item.Name, val, ns.maxBurstPercent)
					return
				}
				ns.namespaceBurstPercent[item.Name] = val
				return
			}
		}
	}
	// annotation was removed from namespace
	delete(ns.namespaceBurstPercent, item.Name)
}

func (ns *NamespaceMgr) deleteNs(item *corev1.Namespace) {
	ns.Lock()
	defer ns.Unlock()
	delete(ns.namespaceBurstPercent, item.Name)

}
