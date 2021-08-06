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
package server

import (
	deviceManager "github.com/bocloud/carina/pkg/devicemanager"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

var dm *deviceManager.DeviceManager
var stopChan chan struct{}

func init() {
	stopChan = make(chan struct{})

	//dm = deviceManager.NewDeviceManager("localhost", c, stopChan)
}

func Start(c echo.Context) error {
	dm.DeviceCheckTask()
	return c.JSON(http.StatusOK, "")
}

func Stop(c echo.Context) error {
	close(stopChan)
	return c.JSON(http.StatusOK, "")
}

func CreateVolume(c echo.Context) error {
	lvName := c.FormValue("lv_name")
	vgName := c.FormValue("vg_name")
	size := c.FormValue("size")
	req, _ := strconv.ParseUint(size, 10, 64)
	err := dm.VolumeManager.CreateVolume(lvName, vgName, req, 1)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "")
}

func ResizeVolume(c echo.Context) error {
	lvName := c.FormValue("lv_name")
	vgName := c.FormValue("vg_name")
	size := c.FormValue("size")
	req, _ := strconv.ParseUint(size, 10, 64)
	err := dm.VolumeManager.ResizeVolume(lvName, vgName, req, 1)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "")
}

func DeleteVolume(c echo.Context) error {
	lvName := c.FormValue("lv_name")
	vgName := c.FormValue("vg_name")
	err := dm.VolumeManager.DeleteVolume(lvName, vgName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "")
}

func GetVolume(c echo.Context) error {
	lvName := c.QueryParam("lv_name")
	vgName := c.QueryParam("vg_name")
	lvInfo, err := dm.VolumeManager.VolumeList(lvName, vgName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, lvInfo)
}

func GetVolumeGroup(c echo.Context) error {

	info, err := dm.VolumeManager.GetCurrentVgStruct()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, info)
}

func CreateSnapshot(c echo.Context) error {
	lvName := c.FormValue("lv_name")
	vgName := c.FormValue("vg_name")
	snapName := c.FormValue("snap_name")
	err := dm.VolumeManager.CreateSnapshot(snapName, lvName, vgName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, "")
}

func DeleteSnapshot(c echo.Context) error {
	vgName := c.FormValue("vg_name")
	snapName := c.FormValue("snap_name")
	err := dm.VolumeManager.DeleteSnapshot(snapName, vgName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, "")
}

func RestoreSnapshot(c echo.Context) error {
	vgName := c.FormValue("vg_name")
	snapName := c.FormValue("snap_name")
	err := dm.VolumeManager.RestoreSnapshot(snapName, vgName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, "")
}

func CloneVolume(c echo.Context) error {
	lvName := c.FormValue("lv_name")
	vgName := c.FormValue("vg_name")
	newLvName := c.FormValue("new_lv_name")
	err := dm.VolumeManager.CloneVolume(lvName, vgName, newLvName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, "")
}
