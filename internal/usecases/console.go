package usecases

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Prepodavan/goca/internal/models"
	"github.com/Prepodavan/goca/internal/utils"
	"github.com/Prepodavan/goca/internal/utils/ctxutils"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const openSSL = "openssl"

type CmdExecutor func(cmd *exec.Cmd) error

type ConsoleUtilWrapper struct {
	debugMode   bool
	rootKeyPath string
	rootCRTPath string
	root        models.Certificate
	executor    CmdExecutor
}

type ConsoleUtilWrapperOption func(*ConsoleUtilWrapper)

func WithExecutor(executor CmdExecutor) ConsoleUtilWrapperOption {
	return func(wrapper *ConsoleUtilWrapper) {
		wrapper.executor = executor
	}
}

func WithDebug(on bool) ConsoleUtilWrapperOption {
	return func(wrapper *ConsoleUtilWrapper) {
		wrapper.debugMode = on
	}
}

func NewConsoleUtilWrapper(keyPath, crtPath string, opts ...ConsoleUtilWrapperOption) (*ConsoleUtilWrapper, error) {
	if err := utils.InitTmpDir(); err != nil {
		return nil, err
	}
	wrapper := &ConsoleUtilWrapper{rootKeyPath: keyPath, rootCRTPath: crtPath}
	if crt, err := os.ReadFile(crtPath); err != nil {
		return nil, err
	} else {
		wrapper.root = models.Certificate{
			Payload: models.Payload{Body: crt},
			Form:    crtPath[strings.LastIndex(crtPath, ".")+1:],
		}
	}
	for _, opt := range opts {
		opt(wrapper)
	}
	return wrapper, nil
}

func (cuw *ConsoleUtilWrapper) Certificate(
	ctx context.Context,
	days uint64,
	conf *models.OpenSSLConfig,
) (*models.Produced, error) {
	newCtx, err := cuw.initRequestContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cuw.cleanUpRequestTmpDir(newCtx)
	tmpDir := cuw.requestTmpDir(newCtx)
	configPath := filepath.Join(tmpDir, "openssl.conf")
	err = os.WriteFile(configPath, conf.Body, 0664)
	if err != nil {
		return nil, err
	}
	keyPath, err := cuw.genPrv(newCtx)
	if err != nil {
		return nil, err
	}
	csrPath, err := cuw.genCSR(newCtx, configPath, keyPath)
	if err != nil {
		return nil, err
	}
	crtPath, err := cuw.genCRT(newCtx, days, csrPath)
	if err != nil {
		return nil, err
	}
	pubKey, err := cuw.genPub(newCtx, crtPath)
	if err != nil {
		return nil, err
	}
	keyBuf, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	crtBuf, err := os.ReadFile(crtPath)
	if err != nil {
		return nil, err
	}
	csrBuf, err := os.ReadFile(csrPath)
	if err != nil {
		return nil, err
	}
	return &models.Produced{
		Certificate:     models.Certificate{Form: "crt", Payload: models.Payload{Body: crtBuf}},
		RootCertificate: cuw.root,
		Request:         models.CSR{Body: csrBuf},
		PrivateKey:      &models.Key{Form: "key", Payload: models.Payload{Body: keyBuf}},
		PublicKey:       *pubKey,
		Config:          conf,
	}, nil
}

func (cuw *ConsoleUtilWrapper) CertificateByCSR(
	ctx context.Context,
	days uint64,
	csr *models.CSR,
) (*models.Produced, error) {
	newCtx, err := cuw.initRequestContext(ctx)
	if err != nil {
		return nil, err
	}
	tmpDir := cuw.requestTmpDir(newCtx)
	defer cuw.cleanUpRequestTmpDir(newCtx)
	csrPath := filepath.Join(tmpDir, "external.csr")
	err = os.WriteFile(csrPath, csr.Body, 0664)
	if err != nil {
		return nil, err
	}
	crtPath, err := cuw.genCRT(newCtx, days, csrPath)
	if err != nil {
		return nil, err
	}
	pubKey, err := cuw.genPub(newCtx, crtPath)
	if err != nil {
		return nil, err
	}
	buf, err := os.ReadFile(crtPath)
	if err != nil {
		return nil, err
	}
	return &models.Produced{
		Certificate:     models.Certificate{Form: "crt", Payload: models.Payload{Body: buf}},
		RootCertificate: cuw.root,
		Request:         *csr,
		PublicKey:       *pubKey,
	}, nil
}

func (cuw *ConsoleUtilWrapper) execute(cmd *exec.Cmd) error {
	if cuw.debugMode {
		// TODO log cmd
		fmt.Println("executing: " + cmd.String())
	}
	if cuw.executor != nil {
		return cuw.executor(cmd)
	}
	return cmd.Run()
}

func (cuw *ConsoleUtilWrapper) cleanUpRequestTmpDir(ctx context.Context) {
	tmpDir := cuw.requestTmpDir(ctx)
	if !cuw.debugMode {
		_ = os.RemoveAll(tmpDir) // TODO mb log
	}
}

func (cuw *ConsoleUtilWrapper) initRequestContext(ctx context.Context) (context.Context, error) {
	requestID := string(ctxutils.MustRequestID(ctx))
	requestTmpDir := filepath.Join(utils.TmpDir(), openSSL, "request:"+requestID)
	info, err := os.Stat(requestTmpDir)
	_ = info
	if err == nil {
		err = os.Remove(requestTmpDir)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(requestTmpDir, 0764); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, "requestTmpDir", requestTmpDir), nil
}

func (cuw *ConsoleUtilWrapper) requestTmpDir(ctx context.Context) string {
	return ctx.Value("requestTmpDir").(string)
}

func (cuw *ConsoleUtilWrapper) genPrv(ctx context.Context) (string, error) {
	outputFilePath := filepath.Join(cuw.requestTmpDir(ctx), "private.key")
	cmd := exec.CommandContext(ctx, openSSL, "genrsa", "-out", outputFilePath)
	if err := cuw.execute(cmd); err != nil {
		return "", err
	}
	return outputFilePath, nil
}

func (cuw *ConsoleUtilWrapper) genCSR(ctx context.Context, configPath, keyPath string) (string, error) {
	out := filepath.Join(cuw.requestTmpDir(ctx), "request.csr")
	cmd := exec.CommandContext(ctx, openSSL, "req", "-new", "-key", keyPath, "-out", out, "-config", configPath)
	if err := cuw.execute(cmd); err != nil {
		return "", err
	}
	return out, nil
}

func (cuw *ConsoleUtilWrapper) genPub(ctx context.Context, crtPath string) (*models.Key, error) {
	pubKeyPath := filepath.Join(cuw.requestTmpDir(ctx), "pub.pem")
	cmd := exec.CommandContext(
		ctx,
		openSSL,
		"x509",
		"-in",
		crtPath,
		"-pubkey",
		"-noout",
	)
	pubKeyBuf := &bytes.Buffer{}
	if !cuw.debugMode {
		cmd.Stdout = pubKeyBuf
	} else {
		pubKeyFile, err := os.OpenFile(pubKeyPath, os.O_RDWR|os.O_CREATE, 0664)
		if err != nil {
			return nil, err
		}
		defer pubKeyFile.Close()
		cmd.Stdout = io.MultiWriter(pubKeyBuf, pubKeyFile)
	}
	if err := cuw.execute(cmd); err != nil {
		return nil, err
	}
	return &models.Key{Payload: models.Payload{Body: pubKeyBuf.Bytes()}, Form: "pem"}, nil
}

func (cuw *ConsoleUtilWrapper) genCRT(ctx context.Context, days uint64, requestPath string) (string, error) {
	out := filepath.Join(cuw.requestTmpDir(ctx), "certificate.crt")
	cmd := exec.CommandContext(
		ctx,
		openSSL,
		"x509",
		"-req",
		"-in",
		requestPath,
		"-days",
		strconv.FormatUint(days, 10),
		"-out",
		out,
		"-CA",
		cuw.rootCRTPath,
		"-CAkey",
		cuw.rootKeyPath,
		"-CAcreateserial",
	)
	if err := cuw.execute(cmd); err != nil {
		return "", err
	}
	return out, nil
}
