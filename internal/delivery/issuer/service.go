package issuer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"github.com/Prepodavan/goca/internal/models"
	"github.com/Prepodavan/goca/internal/utils/ctxutils"
	"github.com/Prepodavan/goca/pkg/issuer/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type UseCase interface {
	Certificate(ctx context.Context, days uint64, config *models.OpenSSLConfig) (*models.Produced, error)
	CertificateByCSR(ctx context.Context, days uint64, csr *models.CSR) (*models.Produced, error)
}

type Service struct {
	useCase UseCase
}

func NewService(useCase UseCase) *Service {
	return &Service{useCase: useCase}
}

func (srv *Service) Apply(router gin.IRouter) {
	router.POST("/cert", srv.bindIssueRequest, srv.issueCertificate, srv.pack)
}

func (srv *Service) bindIssueRequest(ctx *gin.Context) {
	var req dto.IssueRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	} else if req.CSR != nil && req.Config != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "either should be present csr or config",
		})
		return
	}
	ctx.Set("request", &req)
}

func (srv *Service) issueCertificate(ctx *gin.Context) {
	useCaseCtx := ctxutils.WithRequestID(context.TODO(), models.RequestID(uuid.NewString()))
	req := ctx.Value("request").(*dto.IssueRequest)
	var err error
	var output *models.Produced
	conf, csr := &models.OpenSSLConfig{}, &models.CSR{}
	if req.CSR != nil {
		csr.Body, err = srv.readFormFile(req.CSR)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
		output, err = srv.useCase.CertificateByCSR(useCaseCtx, req.Days, csr)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
	} else {
		conf.Body, err = srv.readFormFile(req.Config)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
		output, err = srv.useCase.Certificate(useCaseCtx, req.Days, conf)
		if err != nil {
			srv.internal(ctx, err)
			return
		}

	}
	ctx.Set("produced", output)

}

func (srv *Service) pack(ctx *gin.Context) {
	acceptLower := strings.ToLower(ctx.GetHeader("accept"))
	acceptAll := strings.Contains(acceptLower, "*/*")
	acceptZip := strings.Contains(acceptLower, "application/zip")
	acceptApp := strings.Contains(acceptLower, "application/*")
	acceptTar := strings.Contains(acceptLower, "application/x-tar")
	switch {
	case acceptAll || acceptTar || acceptApp:
		srv.packTar(ctx)
	case acceptZip:
		srv.packZip(ctx)
	default:
		ctx.AbortWithStatus(http.StatusNotAcceptable)
	}
}

func (srv *Service) packZip(ctx *gin.Context) {
	produced := ctx.Value("produced").(*models.Produced)
	buf := &bytes.Buffer{}
	zipper := zip.NewWriter(buf)
	if produced.Config != nil {
		err := srv.createAndWriteZipFile(zipper, "server.conf", produced.Config.Body)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
	}
	if produced.PrivateKey != nil {
		err := srv.createAndWriteZipFile(zipper, "prv.pem", produced.PrivateKey.Body)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
	}
	err := srv.createAndWriteZipFile(zipper, "pub.pem", produced.PublicKey.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.createAndWriteZipFile(zipper, "csr.pem", produced.Request.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.createAndWriteZipFile(zipper, "cert.pem", produced.Certificate.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.createAndWriteZipFile(zipper, "ca-cert.pem", produced.RootCertificate.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = zipper.Close()
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	ctx.Data(http.StatusOK, "application/zip", buf.Bytes())
}

func (srv *Service) createAndWriteZipFile(writer *zip.Writer, name string, body []byte) error {
	file, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(body)
	return err
}

func (srv *Service) packTar(ctx *gin.Context) {
	produced := ctx.Value("produced").(*models.Produced)
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)
	if produced.Config != nil {
		err := srv.writeFileIntoTar(tw, "server.conf", produced.Config.Body)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
	}
	if produced.PrivateKey != nil {
		err := srv.writeFileIntoTar(tw, "prv.pem", produced.PrivateKey.Body)
		if err != nil {
			srv.internal(ctx, err)
			return
		}
	}
	err := srv.writeFileIntoTar(tw, "pub.pem", produced.PublicKey.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.writeFileIntoTar(tw, "csr.pem", produced.Request.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.writeFileIntoTar(tw, "cert.pem", produced.Certificate.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = srv.writeFileIntoTar(tw, "ca-cert.pem", produced.RootCertificate.Body)
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	err = tw.Close()
	if err != nil {
		srv.internal(ctx, err)
		return
	}
	ctx.Data(http.StatusOK, "application/x-tar", buf.Bytes())
}

func (srv *Service) writeFileIntoTar(writer *tar.Writer, name string, body []byte) error {
	hdr := &tar.Header{
		Name: name,
		Size: int64(len(body)),
		Mode: 0600,
	}
	if err := writer.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := writer.Write(body); err != nil {
		return err
	}
	return nil
}

func (srv *Service) internal(ctx *gin.Context, err error) {
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (srv *Service) readFormFile(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
