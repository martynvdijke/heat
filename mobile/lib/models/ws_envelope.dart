class WsEnvelope {
  final String type;
  final String? topic;
  final int seq;
  final dynamic payload;

  const WsEnvelope({required this.type, this.topic, required this.seq, this.payload});

  factory WsEnvelope.fromJson(Map<String, dynamic> j) => WsEnvelope(
        type: j['type'] as String,
        topic: j['topic'] as String?,
        seq: (j['seq'] as num?)?.toInt() ?? 0,
        payload: j['payload'],
      );

  /// Returns true if [incomingSeq] indicates a gap after [lastSeq].
  /// Sequence numbers are monotonically increasing. A gap exists when
  /// incomingSeq > lastSeq + 1.
  static bool hasGap(int lastSeq, int incomingSeq) => incomingSeq > lastSeq + 1;
}
