import 'package:flutter_test/flutter_test.dart';
import 'package:heat/models/racer.dart';
import 'package:heat/models/race_state.dart';
import 'package:heat/models/season.dart';
import 'package:heat/models/round_snapshot.dart';
import 'package:heat/models/standing.dart';
import 'package:heat/models/ws_envelope.dart';

void main() {
  test('Racer fromJson', () {
    final j = {'id':1,'name':'Alice','points':10,'rank':1,'position':2,'car_color':'#ff0000','car_name':'Heat','team_id':3,'team_name':'T','team_color':'#00ff00'};
    final r = Racer.fromJson(j);
    expect(r.id, 1); expect(r.name,'Alice'); expect(r.carColor,'#ff0000'); expect(r.teamId,3);
  });
  test('RaceState fromJson', () {
    final rs = RaceState.fromJson({'state':'racing','elapsed_ms':1234,'current_lap':2,'total_laps':10});
    expect(rs.state,'racing'); expect(rs.elapsedMs,1234); expect(rs.totalLaps,10);
  });
  test('Season fromJson', () {
    final s = Season.fromJson({'id':5,'name':'S1','status':'active','module_ids':[1,2]});
    expect(s.id,5); expect(s.status,'active'); expect(s.moduleIds,[1,2]);
  });
  test('Standing fromJson', () {
    final st = Standing.fromJson({'racer_id':7,'name':'Bob','position':1,'lap':3,'gap':'LEAD'});
    expect(st.racerId,7); expect(st.gap,'LEAD');
  });
  test('RoundSnapshot fromJson', () {
    final j = {'id':1,'season_id':2,'race_name':'GP','round':3,'status':'draft','scores':[{'id':10,'snapshot_id':1,'racer_id':1,'racer_name':'A','points':25,'position':1,'dnf':false,'dns':false,'spins':0,'overheated':false}]};
    final snap = RoundSnapshot.fromJson(j);
    expect(snap.id,1); expect(snap.scores.length,1); expect(snap.scores.first.racerName,'A');
  });
  test('WsEnvelope hasGap', () {
    expect(WsEnvelope.hasGap(5,7), true);
    expect(WsEnvelope.hasGap(5,6), false);
    expect(WsEnvelope.hasGap(5,5), false);
  });
  test('WsEnvelope fromJson', () {
    final e = WsEnvelope.fromJson({'type':'race_state','seq':42,'payload':{'state':'racing','elapsed_ms':0,'current_lap':1,'total_laps':5}});
    expect(e.type,'race_state'); expect(e.seq,42);
  });
}
