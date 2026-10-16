import 'dart:async';
import 'dart:convert';
import 'dart:math';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/ws_envelope.dart';
import 'session_store.dart';

class WsClient {
  final SessionStore store;
  final List<String> topics;
  final void Function(WsEnvelope) onMessage;

  WebSocketChannel? _channel;
  StreamSubscription? _sub;
  Timer? _reconnectTimer;
  int _attempt = 0;
  int _lastSeq = 0;
  bool _closed = false;

  WsClient({required this.store, required this.topics, required this.onMessage});

  int get lastSeq => _lastSeq;

  void connect() {
    _closed = false;
    _doConnect();
  }

  void _doConnect() {
    if (_closed) return;
    final rawBase = store.baseUrl;
    if (rawBase == null || rawBase.isEmpty) return;
    final url = store.wsUrl();
    final token = store.playerToken;
    final protocols = token != null && token.isNotEmpty ? ['heat', 'heat.token.$token'] : ['heat'];
    try {
      _channel = WebSocketChannel.connect(Uri.parse(url), protocols: protocols);
      _sub?.cancel();
      _channel!.sink.add(jsonEncode({'type': 'subscribe', 'topics': topics}));
      _sub = _channel!.stream.listen(
        (data) {
          _attempt = 0;
          final raw = data is String ? data : data.toString();
          try {
            final j = jsonDecode(raw) as Map<String, dynamic>;
            final env = WsEnvelope.fromJson(j);
            if (env.seq != 0) {
              if (WsEnvelope.hasGap(_lastSeq, env.seq)) {
                _channel?.sink.add(jsonEncode({'type': 'resync', 'topics': topics}));
              }
              _lastSeq = mathMax(_lastSeq, env.seq);
            }
            onMessage(env);
          } catch (_) {}
        },
        onError: (_) => _scheduleReconnect(),
        onDone: () => _scheduleReconnect(),
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  int mathMax(int a, int b) => a > b ? a : b;

  void _scheduleReconnect() {
    if (_closed) return;
    _sub?.cancel();
    _attempt++;
    const base = 500;
    const cap = 30000;
    final exp = min(cap, (base * pow(2, _attempt)).toInt());
    final jitter = Random().nextInt(exp + 1);
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(Duration(milliseconds: jitter), _doConnect);
  }

  void dispose() {
    _closed = true;
    _reconnectTimer?.cancel();
    _sub?.cancel();
    _channel?.sink.close();
  }
}
