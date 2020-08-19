package server

import (
	"io"
	"net/http"
	"net/url"
	"os"
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
	db             *mongo.Database
	conf           *config.Config
	clientPximg    *http.Client
	parsedPixivDir string
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

type findOptions struct {
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

	fo := &findOptions{}
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

type aggregateOptions struct {
	Pipeline bson.Raw `json:"pipeline"`
}

func (h *handler) dbAggregate(c echo.Context) error {
	ctx := c.Request().Context()
	ao := &aggregateOptions{}
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

	urlp, err := url.ParseRequestURI(strings.TrimPrefix(req.URL.Path, "/api/v1/proxy/"))
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
	res.Flush()
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
	c.Logger().Info("file ", f, " not found, redirected to proxy ")
	return c.Redirect(http.StatusTemporaryRedirect,
		"/api/v1/proxy/"+u)
}

type userOptions struct {
	Sort  orderedmap.O `json:"sort"`
	Skip  int          `json:"skip"`
	Limit int          `json:"limit"`
}

func (h *handler) user(c echo.Context) error {
	opt := &userOptions{}
	if err := c.Bind(opt); err != nil {
		return err
	}
	ctx := c.Request().Context()
	sort := opt.Sort
	pipeline := append(
		a{d{{Key: "$sort", Value: sort}}},
		model.PipelineUsersAll...)
	pipeline = append(pipeline,
		d{{Key: "$skip", Value: opt.Skip}},
		d{{Key: "$limit", Value: opt.Limit}},
	)
	r, err := h.db.Collection(model.CollectionUser).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	a := []model.Post{}
	err = r.All(ctx, &a)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &a)
}
