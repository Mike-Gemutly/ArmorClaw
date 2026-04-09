rule eicar_test_file {
    meta:
        description = "EICAR standard antivirus test file"
        severity = "high"
    strings:
        $eicar = "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*" wide ascii nocase
    condition:
        $eicar
}

rule macro_pattern {
    meta:
        description = "VBA macro pattern in documents"
        severity = "high"
    strings:
        $macro1 = "AutoOpen" nocase
        $macro2 = "AutoExec" nocase
        $macro3 = "Document_Open" nocase
    condition:
        any of ($macro*)
}
