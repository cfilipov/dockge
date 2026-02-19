use serde::de::{Deserialize, Deserializer, SeqAccess, Visitor};
use serde_json::Value;
use std::fmt;

/// A Vec<Value> wrapper that socketioxide detects as "tuple-like".
///
/// socketioxide's parser probes the `Deserialize` impl to decide how to
/// extract data from a Socket.IO packet:
///   - "tuple-like" types (tuples, fixed arrays) → all packet args are passed
///   - non-tuple types (Vec, structs) → only the first arg is passed
///
/// By implementing `Deserialize` via `deserialize_tuple`, this type tricks
/// the probe into passing all arguments, which we collect into a Vec<Value>.
pub struct SocketArgs(pub Vec<Value>);

impl<'de> Deserialize<'de> for SocketArgs {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ArgsVisitor;

        impl<'de> Visitor<'de> for ArgsVisitor {
            type Value = SocketArgs;

            fn expecting(&self, f: &mut fmt::Formatter) -> fmt::Result {
                write!(f, "a sequence of socket.io arguments")
            }

            fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Self::Value, A::Error> {
                let mut values = Vec::new();
                while let Some(v) = seq.next_element()? {
                    values.push(v);
                }
                Ok(SocketArgs(values))
            }
        }

        // deserialize_tuple triggers socketioxide's is_de_tuple check
        deserializer.deserialize_tuple(16, ArgsVisitor)
    }
}
