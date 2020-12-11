package filters

// TODO: the ccitt doesn't use a ByteReader, so
// we can't be sure we won't read passed EOD

type ccittDecode struct{}
