package api

import (
	"net/http"
	"strings"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type httpRouterHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

func (rt *_router) wrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := reqcontext.RequestContext{
			ReqUUID: reqUUID,
			Logger: rt.baseLogger.WithFields(logrus.Fields{
				"reqid":     reqUUID.String(),
				"remote-ip": r.RemoteAddr,
			}),
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if _, err := rt.db.GetUser(token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx.UserID = token

		fn(w, r, ps, ctx)
	}
}
