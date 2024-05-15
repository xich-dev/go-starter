package controller

import (
	"net/http"
	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/middleware"
	"github.com/xich-dev/go-starter/pkg/service"
)

var phoneRegexp = regexp.MustCompile(`^1[3456789]\d{9}$`)

type Controller struct {
	mid *middleware.Middleware
	svc service.ServiceInterface
}

var _ apigen.ServerInterface = &Controller{}

func NewController(
	s service.ServiceInterface,
	mid *middleware.Middleware,
) *Controller {
	return &Controller{
		svc: s,
		mid: mid,
	}
}

func (a *Controller) GetService() service.ServiceInterface {
	return a.svc
}

func (a *Controller) PostAuthChangePassword(c *fiber.Ctx) error {
	var param apigen.PostAuthChangePasswordJSONBody
	if err := c.BodyParser(&param); err != nil {
		return c.SendStatus(400)
	}
	if !phoneRegexp.MatchString(param.Phone) {
		return c.Status(400).SendString("手机号格式错误")
	}
	if len(param.Code) == 0 {
		return c.Status(400).SendString("验证码不能为空")
	}
	if len(param.NewPassword) == 0 {
		return c.Status(400).SendString("新设密码不能为空")
	}
	if err := a.svc.VerifyCode(c.Context(), param.Phone, apigen.ChangePassword, param.Code); err != nil {
		return c.Status(400).SendString("验证码错误")
	}
	if err := a.svc.ChangePassword(c.Context(), param); err != nil {
		return errors.Wrap(err, "failed to reset password")
	}
	return c.SendStatus(200)
}

func (a *Controller) PostAuthCode(c *fiber.Ctx) error {
	var req apigen.PostAuthCodeJSONBody
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(400)
	}
	if !phoneRegexp.MatchString(req.Phone) {
		return c.Status(400).SendString("手机号格式错误")
	}
	if req.Typ != apigen.Register && req.Typ != apigen.ChangePassword {
		return c.Status(400).SendString("不支持的验证码类型" + string(req.Typ))
	}
	if err := a.svc.CreateCode(c.Context(), apigen.PostAuthCodeJSONBody{
		Phone: req.Phone,
		Typ:   req.Typ,
	}); err != nil {
		if errors.Is(err, service.ErrCodeNotExpired) {
			return c.SendStatus(http.StatusTooManyRequests)
		}
		return errors.Wrap(err, "failed to create code")
	}
	return c.SendStatus(202)
}

func (a *Controller) PostAuthLogin(c *fiber.Ctx) error {
	var req apigen.PostAuthLoginJSONBody
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(400)
	}
	if len(req.UsernameOrPhone) == 0 {
		return c.Status(400).SendString("用户名或手机号不能为空")
	}
	if len(req.Password) == 0 {
		return c.Status(400).SendString("密码不能为空")
	}
	user, rules, err := a.svc.VerifyLoginInfo(c.Context(), req)
	if errors.Is(err, service.ErrUsernameOrPhoneNotFound) {
		return c.Status(404).SendString(err.Error())
	}
	if errors.Is(err, service.ErrIncorrectPassword) {
		return c.Status(403).SendString(err.Error())
	}
	if errors.Is(err, service.ErrDeletedUser) {
		return c.Status(404).SendString(err.Error())
	}
	token, err := a.mid.CreateToken(user, rules)
	if err != nil {
		return err
	}
	authInfo := apigen.AuthInfo{
		Token:     token,
		Id:        &user.ID,
		Username:  user.Name,
		Phone:     user.Phone,
		OrgID:     user.OrgID,
		CreatedAt: &user.CreatedAt,
	}
	return c.Status(200).JSON(authInfo)
}

func (a *Controller) PostAuthLogout(c *fiber.Ctx) error {
	return nil
}

func (a *Controller) GetAuthPing(c *fiber.Ctx) error {
	return nil
}

func (a *Controller) PostAuthRefreshToken(c *fiber.Ctx) error {
	return nil
}

func (a *Controller) PostAuthRegister(c *fiber.Ctx) error {
	var param apigen.PostAuthRegisterJSONBody
	if err := c.BodyParser(&param); err != nil {
		return c.SendStatus(400)
	}
	if !phoneRegexp.MatchString(param.Phone) {
		return c.Status(400).SendString("手机号格式错误")
	}
	if len(param.Code) == 0 {
		return c.Status(400).SendString("验证码不能为空")
	}
	if len(param.Username) == 0 {
		return c.Status(400).SendString("用户名不能为空")
	}
	if len(param.Password) == 0 {
		return c.Status(400).SendString("密码不能为空")
	}
	if err := a.svc.VerifyCode(c.Context(), param.Phone, apigen.Register, param.Code); err != nil {
		return c.Status(400).SendString("验证码错误")
	}
	if err := a.svc.CreateUserWithNewOrg(c.Context(), param); err != nil {
		if errors.Is(err, service.ErrPhoneAlreadyExist) {
			return c.Status(400).SendString("该手机号已被注册")
		}
		if errors.Is(err, service.ErrUsernameAlreadyExist) {
			return c.Status(400).SendString("该用户名已被注册")
		}
		return errors.Wrap(err, "failed to create user with new org")
	}
	return c.SendStatus(200)
}

func (a *Controller) GetOrgs(c *fiber.Ctx) error {
	user, ok := c.Locals(middleware.UserContextKey).(*middleware.User)
	if !ok {
		return c.Status(http.StatusForbidden).SendString("无法从context获取user")
	}

	rtn, err := a.svc.GetOrgInfoByOrgId(c.Context(), user.OrgID)
	if err != nil {
		if errors.Is(err, service.ErrOrgNotFound) {
			return c.Status(404).SendString("没有找到" + user.OrgID.String() + "对应的组织")
		}
		return err
	}
	return c.Status(200).JSON(rtn)
}
