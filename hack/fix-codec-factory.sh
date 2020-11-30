#!/bin/sh

SPELL="type DirectCodecFactory struct {"
SPELL2="	CodecFactory"
SPELL3="}"

TARGET=vendor/k8s.io/apimachinery/pkg/runtime/serializer/codec_factory.go

if grep -Fxq "${SPELL}" ${TARGET}
then
  echo "dependency compatibility is already patched"
else
  # shellcheck disable=SC2028
  echo "\n${SPELL}\n${SPELL2}\n${SPELL3}\n" >> ${TARGET}
fi
