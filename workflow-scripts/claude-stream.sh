#!/usr/bin/env bash
set -euo pipefail

claude "$@" --output-format stream-json --verbose --include-partial-messages -p \
  | jq -nrj --unbuffered '
    def oneline: gsub("\n"; " ⏎ ");
    def trunc($n): if length > $n then .[0:$n] + "…" else . end;

    foreach inputs as $x (
      {ic: 0};

      # state update: track cumulative tool-input bytes within a tool_use block
      if   $x.type == "stream_event"
           and $x.event.type == "content_block_start"
           and $x.event.content_block.type == "tool_use" then .ic = 0
      elif $x.type == "stream_event"
           and ($x.event.delta.type // "") == "input_json_delta" then
        .ic += ($x.event.delta.partial_json | length)
      else . end;

      . as $s |
      ( if $x.type == "stream_event" then
          $x.event as $e |
          if $e.type == "content_block_start" then
            $e.content_block as $cb |
            if   $cb.type == "tool_use"  then "\n[tool_use: \($cb.name)] "
            elif $cb.type == "thinking"  then "\n[thinking]"
            elif $cb.type == "text"      then "\n[text]\n"
            else "" end
          elif $e.type == "content_block_delta" then
            $e.delta as $d |
            if   $d.type == "text_delta"     then $d.text
            elif $d.type == "input_json_delta" then
              ($s.ic - ($d.partial_json | length)) as $prev |
              if $prev >= 80 then ""
              else
                ($d.partial_json | .[0 : 80 - $prev]) as $emit |
                (if $s.ic > 80 and $prev < 80 then $emit + "…" else $emit end)
              end
            else "" end
          elif $e.type == "content_block_stop" then ""
          elif $e.type == "message_stop"       then ""
          else "" end

        elif $x.type == "user" then
          ($x.message.content[0] // {}) as $c |
          if $c.type == "tool_result" then
            ( $c.content
              | if   type == "string" then .
                elif type == "array"  then map(.text // tostring) | join(" ")
                else tostring end
              | oneline | trunc(100)
            ) as $body |
            "\n[tool_result] \($body)"
          else "" end

        elif $x.type == "system" then
          if   $x.subtype == "init"         then "[init session: \($x.model // "")]"
          elif $x.subtype == "notification" then "\n[notification]"
          else "" end

        elif $x.type == "rate_limit_event" then "\n[rate_limit]"
        elif $x.type == "result"           then "\n--- result ---\n" + ($x.result // "")
        else "" end
      )
    )
  '

echo
