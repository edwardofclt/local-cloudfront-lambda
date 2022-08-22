exports.handler = async (event, context, callback) => {
  const { config, response, request } = event.Records[0].cf
  const cookie = getCookie(request?.headers, "eddie-test")
  // console.log(JSON.stringify(event))

  if (config.eventType === "origin-request") {
    return callback(null, {
      headers: {
        ...request.headers,
        "x-origin-request": [
          {
            key: "x-origin-request",
            value: "yep",
          },
        ],
      },
    })
  }

  if (config.eventType == "viewer-response") {
    const responseData = {
      headers: {
        ...response.headers,
        "x-viewer-response": [
          {
            key: "x-viewer-response",
            value: "yep",
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

    return callback(null, {
      headers: {
        ...request.headers,
        "x-viewer-request": [
          {
            key: "x-viewer-request",
            value: "yep",
          },
        ],
      },
    })
  }

  return callback(null, {
    headers: {
      ...response.headers,
      "x-origin-response": [
        {
          key: "x-origin-response",
          value: "yep",
        },
      ],
    },
  })
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
