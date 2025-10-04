package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// ClientReport 客户端上报控制器
type ClientReport struct{}

// FirstInstallReportForm 客户端首次安装上报请求参数
type FirstInstallReportForm struct {
	ClientId     string `json:"client_id" validate:"required"` // 客户端唯一标识
	Hostname     string `json:"hostname"`                      // 客户端主机名
	Platform     string `json:"platform"`                      // 客户端平台（windows/linux/mac/android/ios）
	TargetUserId uint   `json:"target_user_id" validate:"required,gt=0"` // 目标用户ID（需添加到的用户地址簿）
}

// FirstInstallReport 处理客户端首次安装上报
// @Tags 客户端上报
// @Summary 客户端首次安装时上报设备信息
// @Description 接收客户端首次安装时的设备ID等信息，添加到指定用户的当日日期地址簿
// @Accept json
// @Produce json
// @Param body body FirstInstallReportForm true "上报信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/client/first_install [post]
func (cr *ClientReport) FirstInstallReport(c *gin.Context) {
	// 绑定并验证请求参数
	form := &FirstInstallReportForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	// 验证参数有效性
	errList := global.Validator.ValidStruct(c, form)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}

	// 验证目标用户是否存在
	targetUser := service.AllService.UserService.InfoById(form.TargetUserId)
	if targetUser.Id == 0 {
		response.Fail(c, 102, response.TranslateMsg(c, "UserNotFound"))
		return
	}

	// 检查设备是否已在目标用户的地址簿中
	existing := service.AllService.AddressBookService.InfoByUserIdAndId(form.TargetUserId, form.ClientId)
	if existing.RowId > 0 {
		response.Success(c, response.TranslateMsg(c, "DeviceAlreadyExists"))
		return
	}

	// 获取或创建当天日期的地址簿（格式：YYYY-MM-DD）
	loc, _ := time.LoadLocation("Asia/Shanghai") // 按北京时间处理
	dateStr := time.Now().In(loc).Format("2006-01-02")
	defaultCollection, err := service.AllService.AddressBookService.GetOrCreateDateCollection(form.TargetUserId, dateStr)
	if err != nil {
		global.Logger.Error("创建/获取日期地址簿失败: " + err.Error())
		response.Fail(c, 103, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}

	// 创建地址簿条目
	addressBook := &model.AddressBook{
		Id:           form.ClientId,
		Hostname:     form.Hostname,
		Platform:     form.Platform,
		UserId:       form.TargetUserId,
		CollectionId: defaultCollection.Id,
		Status:       0, // 0-离线，1-在线（首次上报默认离线）
	}

	if err := service.AllService.AddressBookService.Create(addressBook); err != nil {
		global.Logger.Error("添加设备到地址簿失败: " + err.Error())
		response.Fail(c, 103, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}

	response.Success(c, response.TranslateMsg(
		c,
		"DeviceAddedSuccessfully",
		strconv.FormatUint(uint64(form.TargetUserId), 10),
	))
}
