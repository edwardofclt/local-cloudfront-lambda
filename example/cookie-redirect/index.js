exports.handler = async (event, context, callback) => {
  const { config, response, request } = event.Records[0].cf
  console.log(config.eventType)
  const cookie = getCookie(request?.headers, "eddie-test")

  if (config.eventType === "origin-request") {
    return callback(null, {
      headers: {
        ...request.headers,
        "x-eddie-test": [
          {
            key: "x-eddie-test",
            value: "EddiesLambda@Edge",
          },
        ],
      },
    })
  }

  if (config.eventType == "viewer-response") {
    const responseData = {
      headers: {
        ...response.headers,
        "x-event-type": [
          {
            key: "x-event-type",
            value: "viewer-response",
          },
        ],
      },
    }

    return callback(null, responseData)
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

  // origin-response
  const responseData = {
    ...(request.uri === "/test-redirect"
      ? {
          status: "302",
        }
      : {}),
    headers: {
      ...response.headers,
      ...(request.uri === "/test-redirect"
        ? {
            location: [
              {
                key: "location",
                value: "https://google.com",
              },
            ],
          }
        : {}),
      "x-event-type": [
        {
          key: "x-event-type",
          value: "viewer-response",
        },
      ],
    },
  }

  console.log(JSON.stringify(responseData))
  return callback(null, responseData)
  // return callback(null, response)
}

const getCookie = (headers, searchFor) => {
  if (headers?.cookie) {
    const cookies = headers.cookie[0].value.split(";")
    for (let i = 0; i < cookies.length; i++) {
      const cookie = cookies[i].split("=")
      if (cookie[0] === searchFor) {
        return cookie[1]
      }
    }
  }
}
