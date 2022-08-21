exports.handler = async (event, context, callback) => {
  const { config, response, request } = event.Records[0].cf
  const cookie = getCookie(request.headers, "eddie-test")

  if (config.eventType === "origin-request") {
    return callback(null, {
      headers: {
        ...request.headers,
        "x-eddie-test": [
          {
            key: "X-Eddie-Test",
            value: "EddiesLambda@Edge",
          },
        ],
      },
    })
  }

  if (config.eventType == "viewer-response") {
    // console.log("viewer-response")
    return callback(null, {
      headers: {
        ...response.headers,
        "x-event-type": [
          {
            key: "x-event-type",
            value: "viewer-response",
          },
        ],
      },
    })
  }

  if (config.eventType === "viewer-request") {
    if (!cookie) {
      const resp = {
        status: "302",
        headers: {
          asdf: [
            {
              key: "asdf",
              value: "yaaaa",
            },
          ],
          location: [
            {
              key: "location",
              value: request.uri,
            },
          ],
          "set-cookie": [
            {
              key: "set-cookie",
              value: `eddie-test=asdf`,
            },
          ],
        },
      }
      return callback(null, resp)
    }

    return callback(null, request)
  }

  return callback(null, response)
}

const getCookie = (headers, searchFor) => {
  if (headers.cookie) {
    const cookies = headers.cookie[0].value.split(";")
    for (let i = 0; i < cookies.length; i++) {
      const cookie = cookies[i].split("=")
      if (cookie[0] === searchFor) {
        return cookie[1]
      }
    }
  }
}
