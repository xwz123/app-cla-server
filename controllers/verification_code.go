package controllers

import (
	"fmt"

	"github.com/opensourceways/app-cla-server/config"
	"github.com/opensourceways/app-cla-server/dbmodels"
	"github.com/opensourceways/app-cla-server/email"
	"github.com/opensourceways/app-cla-server/models"
)

type VerificationCodeController struct {
	baseController
}

func (vc *VerificationCodeController) Prepare() {
	vc.apiPrepare("")
}

// @Title Post
// @Description send verification code when signing
// @Param	:org_cla_id	path 	string					true		"org cla id"
// @Param	:email		path 	string					true		"email of corp"
// @Param   :applyTo    path    string                  true        "corporation  employee or individual"
// @Success 201 {int} map
// @Failure util.ErrSendingEmail
// @router /:link_id/:email/:applyTo [post]
func (vc *VerificationCodeController) Post() {
	action := "create verification code"
	linkID := vc.GetString(":link_id")
	emailOfSigner := vc.GetString(":email")
	applyTo := vc.GetString(":applyTo")

	if applyTo == dbmodels.ApplyToCorporation {
		if err := models.CheckRestrictEmailSuffix(emailOfSigner); err != nil {
			vc.sendModelErrorAsResp(err, action)
			return
		}
	}

	orgInfo, merr := models.GetOrgOfLink(linkID)
	if merr != nil {
		vc.sendFailedResponse(0, "", merr, action)
		return
	}

	code, err := models.CreateVerificationCode(
		emailOfSigner, linkID, config.AppConfig.VerificationCodeExpiry,
	)
	if err != nil {
		vc.sendModelErrorAsResp(err, action)
		return
	}

	vc.sendSuccessResp("create verification code successfully")

	sendEmailToIndividual(
		emailOfSigner, orgInfo.OrgEmail,
		fmt.Sprintf(
			"Verification code for signing CLA on project of \"%s\"",
			orgInfo.OrgAlias,
		),
		email.VerificationCode{
			Email:      emailOfSigner,
			Org:        orgInfo.OrgAlias,
			Code:       code,
			ProjectURL: orgInfo.ProjectURL(),
		},
	)
}

//@Title CodeWithFindPwd
//@Description send verification code when find password
//@Param platform path  string true "code platform"
//@Param org_repo path  string true "org:repo"
//@Param email    query string true "email of contributor"
//@Success 201 {int} map
//@Failure 400 util.ErrSendingEmail
//@router /:platform/:org_repo [get]
func (vc *VerificationCodeController) CodeWithFindPwd() {
	action := "create verification code when find password"
	cmEmail := vc.GetString("email")
	org, repo := parseOrgAndRepo(vc.GetString(":org_repo"))
	linkID, err := models.GetLinkID(buildOrgRepo(vc.GetString(":platform"), org, repo))
	if err != nil {
		vc.sendFailedResponse(400, string(models.ErrNoLinkOrNoManager), err, action)
		return
	}

	orgInfo, err := models.GetOrgOfLink(linkID)
	if err != nil {
		vc.sendFailedResponse(400, string(models.ErrNoLinkOrNoManager), err, action)
		return
	}

	manager, dbErr := dbmodels.GetDB().ListCorporationManager(linkID, cmEmail, "")
	if dbErr != nil || len(manager) == 0 {
		reason := fmt.Errorf("can't find the corporation manager by email")
		vc.sendFailedResponse(400, errUnknownEmailPlatform, reason, action)
		return
	}

	code, err := models.CreateVerificationCode(
		cmEmail, linkID, config.AppConfig.VerificationCodeExpiry,
	)
	if err != nil {
		vc.sendModelErrorAsResp(err, action)
		return
	}

	vc.sendSuccessResp("create verification code successfully")

	sendEmailToIndividual(
		cmEmail, orgInfo.OrgEmail,
		"Verification code for retrieve password ",
		email.FindPasswordVerifyCode{
			Email:      cmEmail,
			Org:        orgInfo.OrgAlias,
			Code:       code,
			ProjectURL: orgInfo.ProjectURL(),
		},
	)
}
