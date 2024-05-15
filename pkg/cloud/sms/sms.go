package sms

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/logger"
)

const FakeCode = "9527"

var (
	smsManager SMSManagerInterface
	log        = logger.NewLogAgent("sms")
)

type SMSManagerInterface interface {
	SendCode(phone string, vcode string) error
	GenerateCode() string
}

type FakeSMSManager struct {
}

func (f *FakeSMSManager) SendCode(phone string, vcode string) error {
	log.Infof("sending %s code %s", phone, vcode)
	return nil
}

func (f *FakeSMSManager) GenerateCode() string {
	return FakeCode
}

type SMSManager struct {
	secretKey     string
	secretId      string
	smsId         string
	smsSigName    string
	smsTemplateId string
	debug         bool
}

func NewSMSManager(cfg *config.Config) SMSManagerInterface {
	if cfg.TCSMS.Enable {
		return &SMSManager{
			secretKey:     cfg.TCSMS.SecretKey,
			secretId:      cfg.TCSMS.SecretId,
			smsId:         cfg.TCSMS.SmsId,
			smsSigName:    cfg.TCSMS.SmsSigName,
			smsTemplateId: cfg.TCSMS.SmsTemplateId,
			debug:         cfg.Debug,
		}
	} else {
		return &FakeSMSManager{}
	}
}

func GetSMSManager() SMSManagerInterface {
	return smsManager
}

func (m *SMSManager) SendCode(phone string, vcode string) error {
	log.Infof("sending to code %s", phone)
	credential := common.NewCredential(
		m.secretId,
		m.secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	// cpf.HttpProfile.ReqTimeout = 5
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"
	// https://cloud.tencent.com/document/api/382/52071#.E5.9C.B0.E5.9F.9F.E5.88.97.E8.A1.A8
	client, _ := sms.NewClient(credential, "ap-guangzhou", cpf)

	request := sms.NewSendSmsRequest()
	request.SmsSdkAppId = common.StringPtr(m.smsId)
	request.SignName = common.StringPtr(m.smsSigName)
	request.TemplateId = common.StringPtr(m.smsTemplateId)
	request.TemplateParamSet = common.StringPtrs([]string{vcode})
	request.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})

	response, err := client.SendSms(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return err
	}
	if err != nil {
		return err
	}
	if m.debug {
		b, _ := json.Marshal(response.Response)
		fmt.Printf("send code %s to %s, res: %s", phone, vcode, string(b))
	}
	return nil
}

func (m *SMSManager) GenerateCode() string {
	return fmt.Sprintf("%d", (1+rand.Intn(10))*10000+rand.Intn(10000))
}
