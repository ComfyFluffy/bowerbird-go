package server

import (
	"crypto/sha1"
	"encoding/hex"
	"image"
	"image/jpeg"
	"strconv"

	// import image decoders
	_ "image/gif"
	_ "image/png"

	"github.com/disintegration/imaging"

	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/WOo0W/bowerbird/config"
	"github.com/WOo0W/bowerbird/helper/orderedmap"
	"github.com/WOo0W/bowerbird/model"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	a = bson.A
	d = bson.D
)

type handler struct {
	db               *mongo.Database
	conf             *config.Config
	clientPximg      *http.Client
	parsedPixivDir   string
	findUserPipeline a
	findPostPipeline a
}

func resultFromCollectionName(collection string) (interface{}, error) {
	var a interface{}
	switch collection {
	case "media":
		a = &[]model.Media{}
	case "posts":
		a = &[]model.Post{}
	case "post_details":
		a = &[]model.PostDetail{}
	case "users":
		a = &[]model.User{}
	case "user_details":
		a = &[]model.UserDetail{}
	case "tags":
		a = &[]model.Tag{}
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "unknown collection: "+collection)
	}
	return a, nil
}

func (h *handler) apiVersion(c echo.Context) error {
	return c.String(200, "bowerbird "+config.Version)
}

func openImage(fullPath string) (image.Image, error) {
	f, err := os.Open(fullPath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, echo.ErrNotFound.SetInternal(err)
		}
		return nil, err
	}
	defer f.Close()
	return imaging.Decode(f)
}

func sendTempThumbnail(c echo.Context, fullPath, name string, width, height int) error {
	tempd := filepath.Join(os.TempDir(), "bowerbird")
	os.MkdirAll(tempd, 0755)

	s := sha1.Sum([]byte(name + strconv.Itoa(width) + "_" + strconv.Itoa(height)))
	tempn := hex.EncodeToString(s[:])
	tempf := filepath.Join(tempd, tempn)

	if _, err := os.Stat(tempf); err == nil {
		return c.File(tempf)
	}

	img, err := openImage(fullPath)
	if err != nil {
		return err
	}
	var w io.Writer
	tempfp := tempf + "_"
	f, err := os.OpenFile(tempfp, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err == nil {
		w = io.MultiWriter(f, c.Response())
		defer f.Close()
	} else {
		w = c.Response()
	}
	img = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	c.Response().Header()["Content-Type"] = []string{"image/jpeg"}
	err = jpeg.Encode(w, img, &jpeg.Options{
		Quality: 75,
	})
	f.Close()
	if err == nil {
		os.Rename(tempfp, tempf)
	}
	return err
}

type localMediaQuery struct {
	Width  int `query:"width"`
	Height int `query:"height"`
}

func (h *handler) localMediaPixiv(c echo.Context) error {
	q := localMediaQuery{}
	if err := c.Bind(&q); err != nil {
		return err
	}
	p, err := url.PathUnescape(c.Param("*"))
	if err != nil {
		return err
	}
	p = path.Clean("/" + p) // "/"+ for security
	fullPath := filepath.Join(h.parsedPixivDir, p)
	if q.Width != 0 && q.Height != 0 {
		return sendTempThumbnail(c, fullPath, p, q.Width, q.Height)
	}
	return c.File(fullPath)
}

type dbFindOptions struct {
	Filter bson.Raw `json:"filter"`
	Skip   *int64   `json:"skip"`
	Limit  *int64   `json:"limit"`
	Sort   bson.Raw `json:"sort"`
}

func (h *handler) dbFind(c echo.Context) error {
	ctx := c.Request().Context()
	collection := c.Param("collection")
	a, err := resultFromCollectionName(collection)
	if err != nil {
		return err
	}

	fo := &dbFindOptions{}
	if err := c.Bind(fo); err != nil {
		return err
	}
	c.Logger().Info("finding "+collection+" ", fo)
	opt := options.Find()
	if len(fo.Sort) > 0 {
		opt.Sort = fo.Sort
	}
	opt.Skip = fo.Skip
	opt.Limit = fo.Limit
	r, err := h.db.Collection(collection).Find(ctx, fo.Filter, opt)
	if err != nil {
		return err
	}
	if err := r.All(ctx, a); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}

type dbAggregateOptions struct {
	Pipeline bson.Raw `json:"pipeline"`
}

func (h *handler) dbAggregate(c echo.Context) error {
	ctx := c.Request().Context()
	ao := &dbAggregateOptions{}
	if err := c.Bind(ao); err != nil {
		return err
	}
	r, err := h.db.Collection(c.Param("collection")).Aggregate(ctx, nil)
	if err != nil {
		return err
	}
	a := &[]map[string]interface{}{}
	err = r.All(ctx, a)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}

func (h *handler) proxy(c echo.Context) error {
	req := c.Request()
	res := c.Response()

	urlp, err := url.ParseRequestURI(c.Param("*"))
	if err != nil {
		return err
	}

	reqProxy, err := http.NewRequestWithContext(req.Context(), req.Method, urlp.String(), nil)
	if err != nil {
		return err
	}

	for k, v := range req.Header {
		if k != "Cookie" &&
			k != "Accept-Encoding" &&
			k != "Host" &&
			k != "Connection" {
			reqProxy.Header[k] = v
		}
	}

	var client *http.Client

	switch {
	case strings.HasSuffix(urlp.Host, ".pximg.net"):
		client = h.clientPximg
		reqProxy.Header["Referer"] = []string{"https://app-api.pixiv.net/"}
	default:
		return echo.NewHTTPError(400, "unsupported host "+urlp.Host)
	}

	reqProxy.URL.RawQuery = req.URL.RawQuery

	resProxy, err := client.Do(reqProxy)
	if err != nil {
		return &echo.HTTPError{
			Code:     http.StatusBadGateway,
			Message:  "cannot make request to " + reqProxy.URL.String(),
			Internal: err,
		}
	}
	defer resProxy.Body.Close()

	for k, v := range resProxy.Header {
		if k != "Content-Encoding" &&
			k != "Set-Cookie" &&
			k != "Transfer-Encoding" {
			res.Header()[k] = v
		}
	}
	res.WriteHeader(resProxy.StatusCode)

	_, err = io.Copy(res, resProxy.Body)
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) mediaByID(c echo.Context) error {
	ctx := c.Request().Context()
	oid, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: err,
		}
	}

	r, err := h.db.Collection(model.CollectionMedia).
		FindOne(ctx, d{{Key: "_id", Value: oid}},
			options.FindOne().SetProjection(
				d{
					{Key: "path", Value: 1},
					{Key: "url", Value: 1},
					{Key: "type", Value: 1},
				},
			)).
		DecodeBytes()
	if err != nil {
		return err
	}
	f, ok := r.Lookup("path").StringValueOK()
	ff := ""
	switch t := model.MediaType(r.Lookup("type").StringValue()); t {
	case model.MediaPixivIllust:
		ff = filepath.Join(h.parsedPixivDir, f)
		f = "pixiv/" + f
	case model.MediaPixivAvatar:
		ff = filepath.Join(h.parsedPixivDir, "avatars", f)
		f = "pixiv/" + f
	case model.MediaPixivProfileBackground:
		ff = filepath.Join(h.parsedPixivDir, "profile_background", f)
		f = "pixiv/" + f
	case model.MediaPixivWorkspaceImage:
		ff = filepath.Join("workspace_images", f)
		f = "pixiv/" + f
	default:
		ff = f
		ok = false
	}
	if ok {
		if _, err := os.Stat(ff); err == nil {
			return c.Redirect(http.StatusTemporaryRedirect,
				"/api/v1/local/"+f)
		}
	}
	u := r.Lookup("url").StringValue()
	c.Logger().Info("file ", f, " not found, redirected to proxy")
	return c.Redirect(http.StatusTemporaryRedirect,
		"/api/v1/proxy/"+u)
}

type findWithPipelineOptions struct {
	Sort  orderedmap.O `json:"sort"`
	Match orderedmap.O `json:"match"`
	Skip  int          `json:"skip"`
	Limit int          `json:"limit"`
}

func findWithPipeline(c echo.Context, collection *mongo.Collection, pipeline1, pipeline2 a, v interface{}) error {
	opt := &findWithPipelineOptions{}
	if err := c.Bind(opt); err != nil {
		return err
	}
	ctx := c.Request().Context()
	sort := opt.Sort
	pipeline := append(
		pipeline1,
		d{{Key: "$sort", Value: sort}},
	)
	pipeline = append(pipeline, pipeline2...)
	pipeline = append(pipeline,
		d{{Key: "$skip", Value: opt.Skip}},
		d{{Key: "$limit", Value: opt.Limit}},
		d{{Key: "$match", Value: opt.Match}},
	)
	r, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	return r.All(ctx, v)
}

func (h *handler) findUser(c echo.Context) error {
	a := &[]model.User{}
	err := findWithPipeline(c, h.db.Collection(model.CollectionUser), h.findUserPipeline, model.PipelineUsersAll, a)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}

func (h *handler) findPost(c echo.Context) error {
	a := &[]model.Post{}
	err := findWithPipeline(c, h.db.Collection(model.CollectionPost), h.findPostPipeline, model.PipelinePostsAll, a)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}
