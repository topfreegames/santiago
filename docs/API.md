Santiago API
============

## Healthcheck Routes

  ### Healthcheck

  `GET /healthcheck`

  Validates that the app is still up, including redis connection.

  * Success Response
    * Code: `200`
    * Content:

      ```
        "WORKING"
      ```

    * Headers:

      It will add an `KHAN-VERSION` header with the current khan module version.

  * Error Response

    It will return an error if it failed to connect to redis.

    * Code: `500`

## Status Routes

  ### Status

  `GET /status`

  Returns statistics on the health of khan.

  * Success Response
    * Code: `200`
    * Content:

      ```
        {
          "app": {
            "errorRate": [float]        // Exponentially Weighted Moving Average Error Rate
          },
          "dispatch": {
            "pendingJobs": [int]        // Pending hook jobs to be sent
          }
        }
      ```

## WebHook Routes

  ### Dispatch webhook
  `POST /hooks`

  Creates a new webhook to be dispatched. This method takes Method and URL as querystring parameters and the payload to send to the webhook as the body.

  * Querystring:

      * `method` - HTTP Method to use to call the webhook (GET, POST, etc);
      * `url` - Endpoint of the webhook to be called.

  * Payload

  The body of this request will be sent without modification to the webhook endpoint.
