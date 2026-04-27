package com.armorclaw.sidecar;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import org.apache.poi.hwpf.HWPFDocument;
import org.apache.poi.hwpf.usermodel.Range;
import org.apache.poi.poifs.filesystem.POIFSFileSystem;
import org.apache.poi.hslf.usermodel.HSLFSlideShow;
import org.apache.poi.hslf.usermodel.HSLFSlide;
import org.apache.poi.hslf.usermodel.HSLFTextBox;

public final class TestFixtures {

    private TestFixtures() {}

    public static byte[] createMinimalDoc() throws IOException {
        POIFSFileSystem fs = new POIFSFileSystem();
        fs.createDocument(new ByteArrayInputStream(new byte[0]), "WordDocument");
        ByteArrayOutputStream seedBaos = new ByteArrayOutputStream();
        fs.writeFilesystem(seedBaos);
        fs.close();

        try (ByteArrayInputStream bais = new ByteArrayInputStream(seedBaos.toByteArray());
             HWPFDocument doc = new HWPFDocument(bais);
             ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            Range range = doc.getRange();
            range.insertBefore("Test document content");
            doc.write(out);
            return out.toByteArray();
        }
    }

    public static byte[] createEmptyDoc() throws IOException {
        POIFSFileSystem fs = new POIFSFileSystem();
        fs.createDocument(new ByteArrayInputStream(new byte[0]), "WordDocument");
        ByteArrayOutputStream seedBaos = new ByteArrayOutputStream();
        fs.writeFilesystem(seedBaos);
        fs.close();

        try (ByteArrayInputStream bais = new ByteArrayInputStream(seedBaos.toByteArray());
             HWPFDocument doc = new HWPFDocument(bais);
             ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            doc.write(out);
            return out.toByteArray();
        }
    }

    public static byte[] createCorruptDoc() {
        return "this is not a valid doc file".getBytes();
    }

    public static byte[] createMinimalPpt() throws IOException {
        try (HSLFSlideShow ppt = new HSLFSlideShow();
             ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            HSLFSlide slide = ppt.createSlide();
            HSLFTextBox textBox = slide.addTitle();
            textBox.setText("Test slide content");
            ppt.write(out);
            return out.toByteArray();
        }
    }

    public static byte[] createEmptyPpt() throws IOException {
        try (HSLFSlideShow ppt = new HSLFSlideShow();
             ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            ppt.createSlide();
            ppt.write(out);
            return out.toByteArray();
        }
    }
}
