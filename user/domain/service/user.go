package service

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"newsCenter/common/jwts"
	"newsCenter/common/unierr"
	"newsCenter/idl/userGrpc"
	"newsCenter/user/domain/entity"
	"newsCenter/user/domain/repository"
	"newsCenter/user/infrastructure/code"
	"newsCenter/user/infrastructure/config"
	"newsCenter/user/infrastructure/persistence/dao"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UserService struct {
	userRepo repository.UserRepository
	cache    repository.Cache
}

func New() *UserService {
	return &UserService{
		userRepo: dao.NewUserDao(),
		cache:    dao.Rc,
	}
}

func (user *UserService) CheckUserNameExits(c context.Context, Username string) (resp *userGrpc.RegisterResponse, err error) {
	UserNameExits, err := user.userRepo.FindUserName(c, Username)
	if err != nil {
		zap.L().Error("CheckUserNameExits FindUserName Fail", zap.Error(err))
		return resp, err
	}
	if UserNameExits {
		zap.L().Error("CheckUserNameExits 用户名存在", zap.Error(err))
		resp = &userGrpc.RegisterResponse{
			StatusCode: unierr.UserNameExist.ErrCode,
			StatusMsg:  unierr.UserNameExist.ErrMsg,
		}
		return resp, err
	}
	return resp, err
}

func (user *UserService) SaveUserInfo(c context.Context, userInfo *entity.UserInfo) (err error) {
	err = user.userRepo.SaveUserInfo(c, userInfo)
	if err != nil {
		zap.L().Error("Register SaveUserInfo Fail", zap.Error(err))
		return err
	}
	return
}
func (user *UserService) CheckUsernameAndPassword(c context.Context, Username string, encryptPassword string) (resp *userGrpc.LoginResponse, userInfo *entity.UserInfo, err error) {
	userInfo, err = user.userRepo.FindUsernameAndPassword(c, Username, encryptPassword)
	if err != nil {
		zap.L().Error("Login FindUsernameAndPassword Fail", zap.Error(err))
		return resp, userInfo, err
	} else if userInfo == nil {
		zap.L().Error("Login userInfo", zap.Error(err))
		resp = &userGrpc.LoginResponse{
			StatusCode: unierr.UsernameOrPasswordErr.ErrCode,
			StatusMsg:  unierr.UsernameOrPasswordErr.ErrMsg,
		}
		return resp, userInfo, nil
	}
	return nil, userInfo, err
}

func (user *UserService) CacheUserInfo(c context.Context, userInfo *entity.UserInfo) (err error) {
	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	//放入缓存 用户Id
	// 声明一个用于传递错误信息的通道
	go func() {
		defer wg.Done()
		marshalUserinfo, err := json.Marshal(userInfo)
		if err != nil {
			errChan <- err
			return
		}
		userIdString := strconv.FormatInt(int64(userInfo.Id), 10)
		expirationTime := time.Duration(config.UserConfig.JwtConfig.AccessExp*3600*24) * time.Second
		err = user.cache.Put(c, code.User+"::"+userIdString, string(marshalUserinfo), expirationTime)
		if err != nil {
			errChan <- err
			return
		}
	}()
	wg.Wait()
	select {
	case err = <-errChan:
		zap.L().Error("Login 用户信息放入缓存失败", zap.Error(err))
		return err
	default:
	}
	return
}

func (user *UserService) ParseToken(token string) (parseToken string, err error) {
	if strings.Contains(token, "bearer ") {
		token = strings.ReplaceAll(token, "bearer ", "")
	}
	parseToken, err = jwts.ParseToken(token, config.UserConfig.JwtConfig.AccessSecret)
	if err != nil {
		zap.L().Error("TokenAuth ParseToken error", zap.Error(err))
		return parseToken, err
	}
	return parseToken, err
}

func (user *UserService) GetCacheUserInfo(c context.Context, parseToken string) (resp *userGrpc.TokenResponse, userInfo *entity.UserInfo, err error) {
	userJson, err := user.cache.Get(c, code.User+"::"+parseToken)
	if err != nil {
		zap.L().Error("TokenAuth cache get user Fail, 用户未登陆 或 被攻击 或 redis崩了 redis崩了结束 防止把数据库也打崩了", zap.Error(err))
		resp = &userGrpc.TokenResponse{
			StatusCode: unierr.NoLogin.ErrCode,
			StatusMsg:  unierr.NoLogin.ErrMsg,
		}
		return resp, nil, err
	}
	//正常是数据库过期了
	//Todo 可能是redis崩了 放过去可能会打崩数据库，先结束。
	//上面保证一定存入，否则一直失败。
	//如果打到不同实例可能会多次登陆，保证每次打在同一实例上。可以用一致性哈希等等
	if userJson == "" {
		zap.L().Error("TokenAuth cache get user expire，过期 或 被攻击")
		resp = &userGrpc.TokenResponse{
			StatusCode: unierr.NoLogin.ErrCode,
			StatusMsg:  unierr.NoLogin.ErrMsg,
		}
		return resp, nil, err
	}
	userInfo = &entity.UserInfo{}
	err = json.Unmarshal([]byte(userJson), userInfo)
	if err != nil {
		zap.L().Error("TokenAuth Unmarshal userJson Fail", zap.Error(err))

		return resp, userInfo, err
	}
	return resp, userInfo, err
}

func CreateToken(userInfo *entity.UserInfo) (token *jwts.JwtToken, err error) {
	userIdString := strconv.FormatInt(int64(userInfo.Id), 10)
	expirationTime := time.Duration(config.UserConfig.JwtConfig.AccessExp*3600*24) * time.Second
	refreshExpirationTime := time.Duration(config.UserConfig.JwtConfig.RefreshExp*3600*24) * time.Second
	token, err = jwts.CreateToken(userIdString, expirationTime, config.UserConfig.JwtConfig.AccessSecret, refreshExpirationTime, config.UserConfig.JwtConfig.RefreshSecret)
	return token, err
}

func (user *UserService) GetUserInfo(c context.Context, userId int64) (userInfo *entity.UserInfo, err error) {
	userInfo, err = user.userRepo.GetUserInfo(c, userId)
	if err != nil {
		zap.L().Error("GetUserInfo GetUserInfo userJson Fail", zap.Error(err))
		return nil, err
	}
	return userInfo, err
}