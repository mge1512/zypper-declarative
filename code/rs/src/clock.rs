// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// A minimal, dependency-free RFC3339 formatter for the wall clock. meta.created_at
// must be a real RFC3339 timestamp (not a small duration formatted as an epoch);
// this converts SystemTime::now() correctly to UTC.

use std::time::{SystemTime, UNIX_EPOCH};

/// Return the current wall-clock time formatted as RFC3339 in UTC, e.g.
/// "2026-06-01T12:34:56Z".
pub fn now_rfc3339() -> String {
    let dur = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    rfc3339_from_unix(dur.as_secs())
}

/// Convert a Unix timestamp (seconds since the epoch, UTC) to RFC3339.
fn rfc3339_from_unix(secs: u64) -> String {
    let days = secs / 86_400;
    let rem = secs % 86_400;
    let hour = rem / 3_600;
    let minute = (rem % 3_600) / 60;
    let second = rem % 60;

    let (year, month, day) = civil_from_days(days as i64);
    format!(
        "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}Z",
        year, month, day, hour, minute, second
    )
}

/// Howard Hinnant's days-to-civil algorithm (proleptic Gregorian, UTC).
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = z - era * 146_097; // [0, 146096]
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365; // [0, 399]
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    let mp = (5 * doy + 2) / 153; // [0, 11]
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32; // [1, 31]
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32; // [1, 12]
    let year = if m <= 2 { y + 1 } else { y };
    (year, m, d)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn known_epoch() {
        // 1970-01-01T00:00:00Z
        assert_eq!(rfc3339_from_unix(0), "1970-01-01T00:00:00Z");
    }

    #[test]
    fn known_date() {
        // 2026-06-01T00:00:00Z = 1780272000
        assert_eq!(rfc3339_from_unix(1_780_272_000), "2026-06-01T00:00:00Z");
    }

    #[test]
    fn now_is_recent_not_epoch() {
        let s = now_rfc3339();
        // It must be a 21st-century date, not a tiny-duration-as-epoch bug.
        assert!(s.starts_with("20"), "created_at must be a real wall-clock date, got {}", s);
    }
}
