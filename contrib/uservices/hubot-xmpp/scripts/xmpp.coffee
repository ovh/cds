
module.exports = (robot) ->
  robot.router.post '/cds/notifications/xmpp', (req, res) ->
    robot.logger.info "data IN"
    event = req.body
    defaultDomain = process.env.HUBOT_XMPP_DEFAULT_DOMAIN

    send = (event, dest) -> 
      robot.logger.info "recipient:#{dest} #{event.subject} #{event.body}"
      if event.subject && event.body
        message = event.subject + '\n' + event.body
      else if event.subject
        message = event.subject
      else if event.body
        message = event.body

      if /@conference/.test dest  
        type = 'groupchat'
        robot.adapter.joinRoom jid: dest
      else if /@/.test dest
        type = 'chat'
      else if /@/.test defaultDomain
        type = 'chat'
        dest = dest + defaultDomain
      else
        robot.logger.info "ignore recipient:#{dest}"
        return

      envelope = 
        room: dest
        user:
          type: type
      robot.logger.info "send to #{dest} (#{envelope.user.type})"
      robot.send(envelope, message)

    send event, dest for dest in event.recipients

    res.writeHead 200, {'Content-Type': 'text/plain'}
    res.end 'Thanks!\n'
