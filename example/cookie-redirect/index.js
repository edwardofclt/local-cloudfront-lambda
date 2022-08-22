exports.handler = async (event, context, callback) => {
  const { config, request } = event.Records[0].cf
  const cookie = getCookie(request?.headers, "eddie-test")

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

  return callback(null, null)
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
