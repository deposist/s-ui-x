<!-- GENERATED from core/capabilities/protocols.json by RenderMatrix.
     Do not edit by hand. -->

# Protocol capability matrix

Single source of truth: `core/capabilities/protocols.json`.

| Type | in | out | group | endpoint | service | tls-tmpl | users | clientDelivery | buildTag | assembledAs | notes/gap |
|---|---|---|---|---|---|---|---|---|---|---|---|
| socks | yes | yes |  |  |  |  | yes | uri |  |  |  |
| http | yes | yes |  |  |  | yes | yes | uri |  |  |  |
| mixed | yes |  |  |  |  |  | yes | uri |  |  |  |
| shadowsocks | yes | yes |  |  |  |  | yes | uri |  |  |  |
| vmess | yes | yes |  |  |  | yes | yes | uri |  |  |  |
| vless | yes | yes |  |  |  | yes | yes | uri |  |  |  |
| trojan | yes | yes |  |  |  | yes | yes | uri |  |  |  |
| naive | yes | yes |  |  |  | yes | yes | uri | with_naive_outbound |  |  |
| hysteria | yes | yes |  |  |  | yes | yes | uri | with_quic |  |  |
| hysteria2 | yes | yes |  |  |  | yes | yes | uri | with_quic |  |  |
| tuic | yes | yes |  |  |  | yes | yes | uri | with_quic |  |  |
| anytls | yes | yes |  |  |  | yes | yes | uri |  |  |  |
| shadowtls | yes | yes |  |  |  |  | yes | broken |  |  | BROKEN delivery: backing-shadowsocks detour is never created; client out_json is a lone shadowtls outbound with no paired shadowsocks. Auto-pair design deferred. |
| direct | yes | yes |  |  |  |  |  | none |  |  |  |
| tun | yes |  |  |  |  |  |  | none |  |  |  |
| redirect | yes |  |  |  |  |  |  | none |  |  |  |
| tproxy | yes |  |  |  |  |  |  | none |  |  |  |
| block |  | yes |  |  |  |  |  |  |  |  |  |
| tor |  | yes |  |  |  |  |  |  |  |  |  |
| ssh |  | yes |  |  |  |  |  |  |  |  |  |
| selector |  |  | yes |  |  |  |  |  |  | selector | Manual operator-selected group backed directly by the core selector outbound. |
| urltest |  |  | yes |  |  |  |  |  |  | urltest | Latency-based group backed directly by the core urltest outbound. |
| failover |  |  | yes |  |  |  |  |  |  | selector | Panel-managed priority failover assembled as a core selector. |
| wireguard |  |  |  | yes |  |  |  |  | with_wireguard |  | Official sing-box wireguard endpoint; warp maps to this type. Requires the upstream wireguard build tag. |
| tailscale |  |  |  | yes |  |  |  |  | with_tailscale |  |  |
| resolved |  |  |  |  | yes |  |  |  |  |  |  |
| ssm-api |  |  |  |  | yes |  |  |  |  |  |  |
| derp |  |  |  |  | yes |  |  |  | with_tailscale |  |  |
