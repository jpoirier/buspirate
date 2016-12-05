/**
 *
 *
 *
 */

#include <SPI.h>

byte inBuf[100];

void setup(void) {
  Serial.begin(115200);
  Serial.println("Configuring the SPI interface for salve mode...");

  pinMode(MOSI, INPUT);
  pinMode(MISO, OUTPUT);
  pinMode(SCK, INPUT);
  pinMode(SS, INPUT);
  digitalWrite(SS, HIGH);

  SPI.beginTransaction(SPISettings(SPI_CLOCK_DIV16, MSBFIRST, SPI_MODE0));

  // turn on SPI in slave mode
  SPCR |= bit(SPE);

  // turn on interrupts
  //SPCR |= bit(SPIE);

  Serial.println("SPI interface configured...");
}

//ISR (SPI_STC_vect) {
//  //Serial.println("isr");
//  SPDR = 0xFF;
//  byte rx = SPDR;
//  Serial.println(rx, DEC);
//}

byte spi_transfer(byte tx) {
  SPDR = tx;
  while (!(SPSR & (1<<SPIF)));
  return SPDR;
}

void spi_transfer_block(void) {
  for (int i = 0; i < 100; i++) {
    while (!(SPSR & (1<<SPIF)));
    inBuf[i] = SPDR;
  }
  for (byte i = 100; i < 200; i++) {
    SPDR = i;
    while(!(SPSR & (1<<SPIF)));
  }
}

byte rx = 0;
byte cnt = 20;

bool block_rcv_mode = false;

void loop() {
  if (!digitalRead(SS)) {
      //Serial.println("LOW");
      if (!block_rcv_mode) {
        rx = spi_transfer(cnt);
        Serial.println(rx, DEC);
        if (rx == 0xFF) {
          Serial.println("going in to block receive mode...");
          block_rcv_mode = true;
          delay(500)
        }
      } else {
        spi_transfer_block();
        for (int i = 0; i < 5; i++) {
          for (int j = 0; j < 20; j++) {
            Serial.print(inBuf[i*20+j], DEC);
            Serial.print(" ");
          }
          Serial.println();
        }
        Serial.println("finished spi_transfer_block...");
        block_rcv_mode = false;
        delay(500)
      }
      cnt += 1;
  }
}
