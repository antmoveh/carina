/*
   Copyright @ 2021 bocloud <fushaosong@beyondcent.com>.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package troubleshoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/anuvu/disko"
	"github.com/carina-io/carina/api"
	carinav1 "github.com/carina-io/carina/api/v1"
	"github.com/carina-io/carina/pkg/devicemanager/partition"
	"github.com/carina-io/carina/pkg/devicemanager/volume"
	"github.com/carina-io/carina/utils"
	"github.com/carina-io/carina/utils/log"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Trouble struct {
	volumeManager volume.LocalVolume
	partition     partition.LocalPartition
	cache         cache.Cache
	nodeName      string
}

const logPrefix = "clean orphan volume:"

func NewTroubleObject(volumeManager volume.LocalVolume, partition partition.LocalPartition, cache cache.Cache, nodeName string) *Trouble {

	if cache == nil {
		return nil
	}

	err := cache.IndexField(context.Background(), &carinav1.LogicVolume{}, "nodeName", func(object client.Object) []string {
		return []string{object.(*carinav1.LogicVolume).Spec.NodeName}
	})

	if err != nil {
		log.Errorf("index node with logicVolume error %s", err.Error())
	}

	return &Trouble{
		volumeManager: volumeManager,
		partition:     partition,
		cache:         cache,
		nodeName:      nodeName,
	}
}

func (t *Trouble) CleanupOrphanVolume() {
	//t.volumeManager.HealthCheck()

	// step.1 获取所有本地volume
	log.Infof("%s get all local logic volume", logPrefix)
	volumeList, err := t.volumeManager.VolumeList("", "")
	if err != nil {
		log.Errorf("% get all local volume failed %s", logPrefix, err.Error())
	}

	// step.2 检查卷状态是否正常
	log.Infof("%s check volume status", logPrefix)
	for _, lv := range volumeList {
		if lv.LVActive != "active" {
			log.Warnf("%s logic volume %s current status %s", logPrefix, lv.LVName, lv.LVActive)
		}
	}

	// step.3 获取集群中logicVolume对象
	log.Infof("%s get all logicVolume in cluster", logPrefix)
	lvList := &carinav1.LogicVolumeList{}
	err = t.cache.List(context.Background(), lvList, client.MatchingFields{"nodeName": t.nodeName})
	if err != nil {
		log.Errorf("%s list logic volume error %s", logPrefix, err.Error())
		return
	}

	// step.4 对比本地volume与logicVolume是否一致， 集群中没有的便删除本地的
	log.Infof("%s cleanup orphan volume", logPrefix)
	mapLvList := map[string]bool{}
	for _, v := range lvList.Items {
		mapLvList[v.Name] = true
		mapLvList[fmt.Sprintf("thin-%s", v.Name)] = true
		mapLvList[fmt.Sprintf("volume-%s", v.Name)] = true
	}

	for _, v := range volumeList {
		if !strings.Contains(v.VGName, "carina") {
			log.Infof("%s skip volume %s", logPrefix, v.LVName)
			continue
		}
		if _, ok := mapLvList[v.LVName]; !ok {
			log.Warnf("%s remove volume %s %s", logPrefix, v.VGName, v.LVName)
			if strings.HasPrefix(v.LVName, "volume-") {
				err := t.volumeManager.DeleteVolume(v.LVName, v.VGName)
				if err != nil {
					log.Errorf("%s delete volume vg %s lv %s error %s", logPrefix, v.VGName, v.LVName, err.Error())
				}
			}
		}
	}

	log.Infof("%s volume check finished.", logPrefix)
}

//清理裸盘分区和logicVolume的对应关系
func (t *Trouble) CleanupOrphanPartition() {
	t.volumeManager.HealthCheck()
	// step.1 获取所有本地 磁盘分区，一个lv其实就是对应一个分区
	log.Infof("%s get all local partition", "CleanupOrphanPartition")
	matchAll := func(d disko.Disk) bool {
		return true
	}
	diskSet, err := t.partition.ScanAllDisks(matchAll)
	if err != nil {
		log.Errorf("fail get all local parttions failed %s", err.Error())
	}

	//TODU step.2 检查磁盘逻辑坏道，物理坏道隔离

	// step.3 获取集群中logicVolume对象
	log.Infof("%s get all logicVolume in cluster", logPrefix)
	lvList := &carinav1.LogicVolumeList{}
	err = t.cache.List(context.Background(), lvList, client.MatchingFields{"nodeName": t.nodeName})
	if err != nil {
		log.Errorf("%s list logic volume error %s", logPrefix, err.Error())
		return
	}

	// step.4 对比本地分区与logicVolume是否一致， 集群中没有的便删除本地磁盘分区
	log.Infof("%s cleanup orphan parttions", logPrefix)
	mapLvList := map[string]bool{}
	for _, v := range lvList.Items {
		if _, ok := v.Annotations[utils.VolumeManagerType]; !ok || v.Annotations[utils.VolumeManagerType] == utils.LvmVolumeType {
			continue
		}
		mapLvList["carina.io/"+v.Name] = true
	}

	for _, disk := range diskSet {
		tmp := api.Disk{}
		utils.Fill(disk, &tmp)
		for _, p := range disk.Partitions {
			if !strings.Contains(p.Name, "carina.io") {
				log.Infof("skip parttions %s", p.Name)
				continue
			}
			if _, ok := mapLvList[p.Name]; !ok {
				log.Warnf("remove parttions %s %s %s", p.Name, p.Start, p.Last)
				if err := t.partition.DeletePartitionByPartNumber(tmp, p.Number); err != nil {
					log.Errorf("%s delete parttions in  %s device %s error %s", disk.Name, p.Number, err.Error())
				}

			}
		}
	}

	log.Infof("%s volume check finished.", logPrefix)
}
